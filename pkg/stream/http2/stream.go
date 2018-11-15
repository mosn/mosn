/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package http2

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/mtls"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	"golang.org/x/net/http2"
)

func init() {
	str.Register(protocol.HTTP2, &streamConnFactory{})
}

type streamConnFactory struct{}

func (f *streamConnFactory) CreateClientStream(context context.Context, connection types.ClientConnection,
	streamConnCallbacks types.StreamConnectionEventListener, connCallbacks types.ConnectionEventListener) types.ClientStreamConnection {
	return newClientStreamConnection(context, connection, streamConnCallbacks, connCallbacks)
}

func (f *streamConnFactory) CreateServerStream(context context.Context, connection types.Connection,
	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	return newServerStreamConnection(context, connection, callbacks)
}

func (f *streamConnFactory) CreateBiDirectStream(context context.Context, connection types.ClientConnection,
	clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {
	return nil
}

var server http2.Server

// types.StreamConnection
// types.StreamConnectionEventListener
type streamConnection struct {
	context context.Context

	protocol      types.Protocol
	connection    types.Connection
	http2Conn     *http2.ClientConn
	asMutex       sync.Mutex
	connCallbacks types.ConnectionEventListener

	logger log.Logger
}

// types.StreamConnection
func (conn *streamConnection) Dispatch(buffer types.IoBuffer) {
}

func (conn *streamConnection) Protocol() types.Protocol {
	return conn.protocol
}

func (conn *streamConnection) GoAway() {
	// todo
}

// types.ClientStreamConnection
type clientStreamConnection struct {
	streamConnection
	streamConnCallbacks types.StreamConnectionEventListener
}

func newClientStreamConnection(context context.Context, connection types.ClientConnection,
	streamConnCallbacks types.StreamConnectionEventListener,
	connCallbacks types.ConnectionEventListener) types.ClientStreamConnection {

	log.DefaultLogger.Tracef("new http2 client stream connection")
	return &clientStreamConnection{
		streamConnection: streamConnection{
			context:       context,
			connection:    connection,
			http2Conn:     context.Value(H2ConnKey).(*http2.ClientConn),
			connCallbacks: connCallbacks,
			logger:        log.ByContext(context),
		},
		streamConnCallbacks: streamConnCallbacks,
	}
}

func (csc *clientStreamConnection) OnGoAway() {
	csc.streamConnCallbacks.OnGoAway()
}

func (csc *clientStreamConnection) NewStream(ctx context.Context, streamID string, responseDecoder types.StreamReceiver) types.StreamSender {
	log.DefaultLogger.Tracef("http2 client stream connection new stream , stream id = %v", streamID)
	stream := &clientStream{
		stream: stream{
			context: context.WithValue(ctx, types.ContextKeyStreamID, streamID),
			decoder: responseDecoder,
		},
		connection: csc,
	}

	return stream
}

// types.ServerStreamConnection
type serverStreamConnection struct {
	streamConnection
	serverStreamConnCallbacks types.ServerStreamConnectionEventListener
}

func newServerStreamConnection(context context.Context, connection types.Connection,
	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	log.DefaultLogger.Tracef("new http2 server stream connection")
	connection.SetReadDisable(true)
	ssc := &serverStreamConnection{
		streamConnection: streamConnection{
			context:    context,
			connection: connection,
		},
		serverStreamConnCallbacks: callbacks,
	}

	if tlsConn, ok := ssc.connection.RawConn().(*mtls.TLSConn); ok {
		tlsConn.SetALPN(http2.NextProtoTLS)
		if err := tlsConn.Handshake(); err != nil {
			logger := log.ByContext(context)
			logger.Errorf("TLS handshake error from %s: %v", ssc.connection.RemoteAddr(), err)
			return nil
		}
	}

	server.ServeConn(connection.RawConn(), &http2.ServeConnOpts{
		Handler: ssc,
	})

	return ssc
}

func (ssc *serverStreamConnection) OnGoAway() {
	ssc.serverStreamConnCallbacks.OnGoAway()
}

func (ssc *serverStreamConnection) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	//generate stream id using global counter
	streamID := protocol.GenerateIDString()

	stream := &serverStream{
		stream: stream{
			context: context.WithValue(ssc.context, types.ContextKeyStreamID, streamID),
			request: request,
		},
		connection:       ssc,
		responseWriter:   responseWriter,
		responseDoneChan: make(chan struct{}),
	}
	stream.decoder = ssc.serverStreamConnCallbacks.NewStream(stream.stream.context, streamID, stream)

	if atomic.LoadInt32(&stream.readDisableCount) <= 0 {
		defer func() {
			if rec := recover(); rec != nil {
				stream.responseWriter = nil
			}
		}()

		stream.handleRequest()
	}

	<-stream.responseDoneChan
}

// types.Stream
// types.StreamSender
type stream struct {
	context context.Context

	readDisableCount int32
	request          *http.Request
	response         *http.Response
	decoder          types.StreamReceiver
	streamCbs        []types.StreamEventListener
}

// types.Stream
func (s *stream) AddEventListener(streamCb types.StreamEventListener) {
	s.streamCbs = append(s.streamCbs, streamCb)
}

func (s *stream) RemoveEventListener(streamCb types.StreamEventListener) {
	cbIdx := -1

	for i, streamCb := range s.streamCbs {
		if streamCb == streamCb {
			cbIdx = i
			break
		}
	}

	if cbIdx > -1 {
		s.streamCbs = append(s.streamCbs[:cbIdx], s.streamCbs[cbIdx+1:]...)
	}
}

func (s *stream) ResetStream(reason types.StreamResetReason) {
	for _, cb := range s.streamCbs {
		cb.OnResetStream(reason)
	}
}

type clientStream struct {
	stream
	connection *clientStreamConnection
}

// types.StreamSender
func (s *clientStream) AppendHeaders(context context.Context, headers types.HeaderMap, endStream bool) error {
	log.DefaultLogger.Tracef("http2 client stream encode headers")
	headersMap, _ := headers.(protocol.CommonHeader)
	scheme := "http"
	if _, ok := s.connection.connection.RawConn().(*mtls.TLSConn); ok {
		scheme = "https"
	}
	if s.request == nil {
		s.request = new(http.Request)
		// TODO: protocol convert in protocol
		// if the request contains body, use "Put" as default, the http request method will be setted by MosnHeaderMethod
		if endStream {
			s.request.Method = http.MethodGet
		} else {
			s.request.Method = http.MethodPost
		}
		s.request.URL, _ = url.Parse(fmt.Sprintf(scheme+"://%s/",
			s.connection.connection.RemoteAddr().String()))
	}
	if method, ok := headersMap[protocol.MosnHeaderMethod]; ok {
		s.request.Method = method
		delete(headersMap, protocol.MosnHeaderMethod)
	}
	if host, ok := headersMap[protocol.MosnHeaderHostKey]; ok {
		s.request.Host = host
		delete(headersMap, protocol.MosnHeaderHostKey)
	}
	var URI string

	if path, ok := headersMap[protocol.MosnHeaderPathKey]; ok {
		URI = fmt.Sprintf(scheme+"://%s%s", s.connection.connection.RemoteAddr().String(), path)
		delete(headersMap, protocol.MosnHeaderPathKey)
	}

	if URI != "" {

		if queryString, ok := headersMap[protocol.MosnHeaderQueryStringKey]; ok {
			URI += "?" + queryString
		}

		s.request.URL, _ = url.Parse(URI)
	}

	// delete inner header
	if _, ok := headersMap[protocol.MosnHeaderQueryStringKey]; ok {
		delete(headersMap, protocol.MosnHeaderQueryStringKey)
	}

	s.request.Header = encodeHeader(headersMap)

	log.DefaultLogger.Tracef("http2 client stream encode headers,headers = %v", s.request.Header)

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *clientStream) AppendData(context context.Context, data types.IoBuffer, endStream bool) error {
	log.DefaultLogger.Tracef("http2 client stream encode data")
	if s.request == nil {
		s.request = new(http.Request)
	}

	s.request.Body = &IoBufferReadCloser{
		buf: data,
	}

	log.DefaultLogger.Tracef("http2 client stream encode data,data = %v", data.String())

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *clientStream) AppendTrailers(context context.Context, trailers types.HeaderMap) error {
	log.DefaultLogger.Tracef("http2 client stream encode trailers")
	s.request.Trailer = encodeHeader(trailers.(protocol.CommonHeader))
	s.endStream()

	return nil
}

func (s *clientStream) endStream() {
	go s.doSend()
}

func (s *clientStream) ReadDisable(disable bool) {
	if disable {
		atomic.AddInt32(&s.readDisableCount, 1)
	} else {
		newCount := atomic.AddInt32(&s.readDisableCount, -1)

		if newCount <= 0 {
			s.handleResponse()
		}
	}
}

func (s *clientStream) doSend() {
	resp, err := s.connection.http2Conn.RoundTrip(s.request)

	if err != nil {
		// due to we use golang h2 conn impl, we need to do some adapt to some things observable
		switch err.(type) {
		case http2.StreamError:
			s.ResetStream(types.StreamRemoteReset)
			s.CleanStream()
		case http2.GoAwayError:
			s.connection.streamConnCallbacks.OnGoAway()
			s.CleanStream()
		case error:
			// todo: target remote close event
			if err == io.EOF {
				s.connection.connection.Close(types.NoFlush, types.RemoteClose)
			} else if err.Error() == "http2: client conn is closed" {
				// we don't use mosn io impl, so get connection state from golang h2 io read/write loop
				s.connection.connection.Close(types.NoFlush, types.LocalClose)
			} else if err.Error() == "http2: client conn not usable" {
				// raise overflow event to let conn pool taking action
				s.ResetStream(types.StreamOverflow)
			} else if err.Error() == "http2: Transport received Server's graceful shutdown GOAWAY" {
				s.connection.streamConnCallbacks.OnGoAway()
			} else if err.Error() == "http2: Transport received Server's graceful shutdown GOAWAY; some request body already written" {
				// todo: retry
			} else if err.Error() == "http2: timeout awaiting response headers" {
				s.ResetStream(types.StreamConnectionFailed)
			} else if err.Error() == "net/http: request canceled" {
				s.ResetStream(types.StreamLocalReset)
			} else {
				s.connection.logger.Errorf("Unknown err: %v", err)
				s.connection.connection.Close(types.NoFlush, types.RemoteClose)
			}

			s.CleanStream()
		}
	} else {
		s.response = resp
		if atomic.LoadInt32(&s.readDisableCount) <= 0 {
			s.handleResponse()
		}
	}
}

func (s *clientStream) CleanStream() {
	s.connection.asMutex.Lock()
	s.response = nil
	s.connection.asMutex.Unlock()
}

func (s *clientStream) handleResponse() {
	if s.response != nil {
		s.decoder.OnReceiveHeaders(s.context, protocol.CommonHeader(decodeRespHeader(s.response)), false)
		buf := buffer.NewIoBuffer(1024)
		buf.ReadFrom(s.response.Body)
		s.decoder.OnReceiveData(s.context, buf, false)
		s.decoder.OnReceiveTrailers(s.context, protocol.CommonHeader(decodeHeader(s.response.Trailer)))

		s.connection.asMutex.Lock()
		s.response = nil
		s.connection.asMutex.Unlock()
	}
}

func (s *clientStream) GetStream() types.Stream {
	return s
}

type serverStream struct {
	stream
	connection       *serverStreamConnection
	responseWriter   http.ResponseWriter
	responseDone     uint32
	responseDoneChan chan struct{}
}

// types.StreamSender
func (s *serverStream) AppendHeaders(context context.Context, headersIn types.HeaderMap, endStream bool) error {
	headers := headersIn.(protocol.CommonHeader)

	if s.response == nil {
		s.response = new(http.Response)
	}

	if value, ok := headers[types.HeaderStatus]; ok {
		s.response.StatusCode, _ = strconv.Atoi(value)
		delete(headers, types.HeaderStatus)
	} else {
		s.response.StatusCode = 200
	}

	s.response.Header = encodeHeader(headers)

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *serverStream) AppendData(context context.Context, data types.IoBuffer, endStream bool) error {
	if s.response == nil {
		s.response = new(http.Response)
	}

	s.response.Body = &IoBufferReadCloser{
		buf: data,
	}

	if endStream {
		s.endStream()
	}
	return nil
}

func (s *serverStream) AppendTrailers(context context.Context, trailers types.HeaderMap) error {
	s.response.Trailer = encodeHeader(trailers.(protocol.CommonHeader))

	s.endStream()
	return nil
}

func (s *serverStream) endStream() {
	s.doSend()

	if atomic.CompareAndSwapUint32(&s.responseDone, 0, 1) {
		close(s.responseDoneChan)
	}
}

func (s *serverStream) ResetStream(reason types.StreamResetReason) {
	// on stream reset
	if atomic.CompareAndSwapUint32(&s.responseDone, 0, 1) {
		close(s.responseDoneChan)
	}

	s.stream.ResetStream(reason)
}

func (s *serverStream) ReadDisable(disable bool) {
	if disable {
		atomic.AddInt32(&s.readDisableCount, 1)
	} else {
		newCount := atomic.AddInt32(&s.readDisableCount, -1)

		if newCount <= 0 {
			s.handleRequest()
		}
	}
}

func (s *serverStream) doSend() {
	if s.responseWriter == nil {
		s.connection.logger.Warnf("responseWriter is nil, send stream is ignored")

		return
	}

	for key, values := range s.response.Header {
		for _, value := range values {
			s.responseWriter.Header().Add(key, value)
		}
	}

	s.responseWriter.WriteHeader(s.response.StatusCode)

	if s.response.Body != nil {
		buf := buffer.NewIoBuffer(1024)
		buf.ReadFrom(s.response.Body)
		buf.WriteTo(s.responseWriter)
	}
}

func (s *serverStream) handleRequest() {
	if s.request != nil {

		header := decodeHeader(s.request.Header)

		// set host header if not found, just for insurance
		if _, ok := header[protocol.MosnHeaderHostKey]; !ok {
			header[protocol.MosnHeaderHostKey] = s.request.Host
		}

		// set path header if not found
		path, queryString := parsePathFromURI(s.request.RequestURI)

		if _, ok := header[protocol.MosnHeaderPathKey]; !ok {
			header[protocol.MosnHeaderPathKey] = string(path)
		}

		// set query string header if not found
		if _, ok := header[protocol.MosnHeaderQueryStringKey]; !ok {
			header[protocol.MosnHeaderQueryStringKey] = string(queryString)
		}
		// set method string header if not found
		if _, ok := header[protocol.MosnHeaderMethod]; !ok {
			header[protocol.MosnHeaderMethod] = s.request.Method
		}

		s.decoder.OnReceiveHeaders(s.context, protocol.CommonHeader(header), false)

		//remove detect
		//if s.element != nil {
		buf := buffer.NewIoBuffer(1024)
		buf.ReadFrom(s.request.Body)
		s.decoder.OnReceiveData(s.context, buf, false)
		s.decoder.OnReceiveTrailers(s.context, protocol.CommonHeader(decodeHeader(s.request.Trailer)))
		//}
	}
}

func (s *serverStream) GetStream() types.Stream {
	return s
}

func encodeHeader(in map[string]string) (out map[string][]string) {
	out = make(map[string][]string, len(in))

	for k, v := range in {
		out[k] = strings.Split(v, ",")
	}

	return
}

func decodeHeader(in map[string][]string) (out map[string]string) {
	out = make(map[string]string, len(in))

	for k, v := range in {
		//// convert to lower case for internal process
		out[strings.ToLower(k)] = strings.Join(v, ",")
	}

	// inherit upstream's response status
	return
}

func decodeRespHeader(resp *http.Response) (out map[string]string) {
	header := decodeHeader(resp.Header)

	// inherit upstream's response status
	header[types.HeaderStatus] = strconv.Itoa(resp.StatusCode)

	return header
}

// io.ReadCloser
type IoBufferReadCloser struct {
	buf types.IoBuffer
}

func (rc *IoBufferReadCloser) Read(p []byte) (n int, err error) {
	return rc.buf.Read(p)
}

func (rc *IoBufferReadCloser) Close() error {
	rc.buf.Reset()
	return nil
}

// GET /rest/1.0/file?fields=P_G&bz=test
// return path and query string
func parsePathFromURI(requestURI string) (string, string) {

	if "" == requestURI {
		return "", ""
	}

	queryMaps := strings.Split(requestURI, "?")
	if len(queryMaps) > 1 {
		return queryMaps[0], queryMaps[1]
	}
	return queryMaps[0], ""

}
