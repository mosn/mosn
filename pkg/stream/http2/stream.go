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
	"container/list"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network/buffer"
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

var transport http2.Transport
var server http2.Server

// types.StreamConnection
// types.StreamConnectionEventListener
type streamConnection struct {
	context context.Context

	protocol      types.Protocol
	rawConnection net.Conn
	http2Conn     *http2.ClientConn
	activeStreams *list.List
	asMutex       sync.Mutex
	connCallbacks types.ConnectionEventListener

	logger log.Logger
}

// types.StreamConnection
func (conn *streamConnection) Dispatch(buffer types.IoBuffer) {}

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

	h2Conn, _ := transport.NewClientConn(connection.RawConn())

	return &clientStreamConnection{
		streamConnection: streamConnection{
			context:       context,
			rawConnection: connection.RawConn(),
			http2Conn:     h2Conn,
			activeStreams: list.New(),
			connCallbacks: connCallbacks,
		},
		streamConnCallbacks: streamConnCallbacks,
	}
}

func (csc *clientStreamConnection) OnGoAway() {
	csc.streamConnCallbacks.OnGoAway()
}

func (csc *clientStreamConnection) NewStream(streamId string, responseDecoder types.StreamReceiver) types.StreamSender {
	stream := &clientStream{
		stream: stream{
			context: context.WithValue(csc.context, types.ContextKeyStreamID, streamId),
			decoder: responseDecoder,
		},
		connection: csc,
	}

	csc.asMutex.Lock()
	stream.element = csc.activeStreams.PushBack(stream)
	csc.asMutex.Unlock()

	return stream
}

// types.ServerStreamConnection
type serverStreamConnection struct {
	streamConnection
	serverStreamConnCallbacks types.ServerStreamConnectionEventListener
}

func newServerStreamConnection(context context.Context, connection types.Connection,
	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	connection.SetReadDisable(true)
	ssc := &serverStreamConnection{
		streamConnection: streamConnection{
			context:       context,
			rawConnection: connection.RawConn(),
			activeStreams: list.New(),
		},
		serverStreamConnCallbacks: callbacks,
	}
	if tlsConn, ok := ssc.rawConnection.(*tls.Conn); ok {

		if err := tlsConn.Handshake(); err != nil {
			logger := log.ByContext(context)
			logger.Errorf("TLS handshake error from %s: %v", ssc.rawConnection.RemoteAddr(), err)
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

//作为PROXY的STREAM SERVER
func (ssc *serverStreamConnection) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	//generate stream id using timestamp
	streamId := "streamID-" + time.Now().String()

	stream := &serverStream{
		stream: stream{
			context: context.WithValue(ssc.context, types.ContextKeyStreamID, streamId),
			request: request,
		},
		connection:       ssc,
		responseWriter:   responseWriter,
		responseDoneChan: make(chan bool, 1),
	}
	stream.decoder = ssc.serverStreamConnCallbacks.NewStream(streamId, stream)
	ssc.asMutex.Lock()
	stream.element = ssc.activeStreams.PushBack(stream)
	ssc.asMutex.Unlock()

	if atomic.LoadInt32(&stream.readDisableCount) <= 0 {
		stream.handleRequest()
	}

	select {
	case <-stream.responseDoneChan:
	}
}

// types.Stream
// types.StreamSender
type stream struct {
	context context.Context

	readDisableCount int32
	request          *http.Request
	response         *http.Response
	decoder          types.StreamReceiver
	element          *list.Element
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
func (s *clientStream) AppendHeaders(headers interface{}, endStream bool) error {
	log.StartLogger.Tracef("http2 client stream encode headers")
	headersMap, _ := headers.(map[string]string)

	if s.request == nil {
		s.request = new(http.Request)
		s.request.Method = http.MethodGet
		s.request.URL, _ = url.Parse(fmt.Sprintf("http://%s/",
			s.connection.rawConnection.RemoteAddr().String()))
	}

	if method, ok := headersMap[types.HeaderMethod]; ok {
		s.request.Method = method
		delete(headersMap, types.HeaderMethod)
	}

	if host, ok := headersMap[types.HeaderHost]; ok {
		s.request.Host = host
		delete(headersMap, types.HeaderHost)
	}

	if path, ok := headersMap[protocol.MosnHeaderPathKey]; ok {
		s.request.URL, _ = url.Parse(fmt.Sprintf("http://%s%s",
			s.connection.rawConnection.RemoteAddr().String(), path))
		delete(headersMap, protocol.MosnHeaderPathKey)
	}

	if _, ok := headersMap["Host"]; ok {
		headersMap["Host"] = s.connection.rawConnection.RemoteAddr().String()
		s.request.Host = s.connection.rawConnection.RemoteAddr().String()
	}

	s.request.Header = encodeHeader(headersMap)

	log.StartLogger.Tracef("http2 client stream encode headers,headers = %v", s.request.Header)

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *clientStream) AppendData(data types.IoBuffer, endStream bool) error {
	log.StartLogger.Tracef("http2 client stream encode data")
	if s.request == nil {
		s.request = new(http.Request)
	}

	s.request.Method = http.MethodPost
	s.request.Body = &IoBufferReadCloser{
		buf: data,
	}

	log.StartLogger.Tracef("http2 client stream encode data,data = %v", data.String())

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *clientStream) AppendTrailers(trailers map[string]string) error {
	log.StartLogger.Tracef("http2 client stream encode trailers")
	s.request.Trailer = encodeHeader(trailers)
	s.endStream()

	return nil
}

func (s *clientStream) endStream() {
	s.doSend()
}

func (s *clientStream) ReadDisable(disable bool) {
	s.connection.logger.Debugf("high watermark on h2 stream client")

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
		log.StartLogger.Tracef("http2 client stream send error %v", err)
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
				s.connection.connCallbacks.OnEvent(types.RemoteClose)
			} else if err.Error() == "http2: client conn is closed" {
				// we dont use mosn io impl, so get connection state from golang h2 io read/write loop
				s.connection.connCallbacks.OnEvent(types.LocalClose)
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
				s.ResetStream(types.StreamConnectionFailed)
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
	s.connection.activeStreams.Remove(s.element)
	s.connection.asMutex.Unlock()
}

func (s *clientStream) handleResponse() {
	if s.response != nil {
		s.decoder.OnReceiveHeaders(decodeHeader(s.response.Header), false)
		buf := &buffer.IoBuffer{}
		buf.ReadFrom(s.response.Body)
		s.decoder.OnReceiveData(buf, false)
		s.decoder.OnReceiveTrailers(decodeHeader(s.response.Trailer))

		s.connection.asMutex.Lock()
		s.response = nil
		s.connection.activeStreams.Remove(s.element)
		s.element = nil
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
	responseDoneChan chan bool
}

// types.StreamSender
func (s *serverStream) AppendHeaders(headersIn interface{}, endStream bool) error {
	headers, _ := headersIn.(map[string]string)

	if s.response == nil {
		s.response = new(http.Response)
		s.response.StatusCode = 200
	}

	s.response.Header = encodeHeader(headers)

	if status := s.response.Header.Get(types.HeaderStatus); status != "" {
		s.response.StatusCode, _ = strconv.Atoi(status)
		s.response.Header.Del(types.HeaderStatus)
	}

	if endStream {
		s.endStream()
	}
	return nil
}

func (s *serverStream) AppendData(data types.IoBuffer, endStream bool) error {
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

func (s *serverStream) AppendTrailers(trailers map[string]string) error {
	s.response.Trailer = encodeHeader(trailers)

	s.endStream()
	return nil
}

func (s *serverStream) endStream() {
	s.doSend()
	s.responseDoneChan <- true

	s.connection.asMutex.Lock()
	s.connection.activeStreams.Remove(s.element)
	s.element = nil
	s.connection.asMutex.Unlock()
}

func (s *serverStream) ReadDisable(disable bool) {
	s.connection.logger.Debugf("high watermark on h2 stream server")

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
	for key, values := range s.response.Header {
		for _, value := range values {
			s.responseWriter.Header().Add(key, value)
		}
	}

	s.responseWriter.WriteHeader(s.response.StatusCode)

	if s.response.Body != nil {
		buf := &buffer.IoBuffer{}
		buf.ReadFrom(s.response.Body)
		buf.WriteTo(s.responseWriter)
	}
}

func (s *serverStream) handleRequest() {
	if s.request != nil {
		s.decoder.OnReceiveHeaders(decodeHeader(s.request.Header), false)

		//remove detect
		if s.element != nil {
			buf := &buffer.IoBuffer{}
			buf.ReadFrom(s.request.Body)
			s.decoder.OnReceiveData(buf, false)
			s.decoder.OnReceiveTrailers(decodeHeader(s.request.Trailer))
		}
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

	return
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
