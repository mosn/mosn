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

package http

import (
	"container/list"
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/valyala/fasthttp"
)

func init() {
	str.Register(protocol.HTTP1, &streamConnFactory{})
}

type streamConnFactory struct{}

func (f *streamConnFactory) CreateClientStream(context context.Context, connection types.ClientConnection,
	streamConnCallbacks types.StreamConnectionEventListener, connCallbacks types.ConnectionEventListener) types.ClientStreamConnection {
	return nil
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

// types.StreamConnection
// types.StreamConnectionEventListener
type streamConnection struct {
	context context.Context

	protocol      types.Protocol
	rawConnection net.Conn
	activeStream  *stream //concurrent stream not supported in HTTP/1.X, so list storage for active stream is not required
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

// since we use fasthttp client, which already implements the conn-pool feature and manage the connections itself
// there is no need to do the connection management. so http/1.x stream only wrap the progress for constructing request/response
// and have no aware of the connection it would use

// types.ClientStreamConnection
type clientStreamWrapper struct {
	context context.Context

	client *fasthttp.HostClient

	activeStreams *list.List
	asMutex       sync.Mutex

	connCallbacks       types.ConnectionEventListener
	streamConnCallbacks types.StreamConnectionEventListener
}

func newClientStreamWrapper(context context.Context, client *fasthttp.HostClient,
	streamConnCallbacks types.StreamConnectionEventListener,
	connCallbacks types.ConnectionEventListener) types.ClientStreamConnection {

	return &clientStreamWrapper{
		context:             context,
		client:              client,
		activeStreams:       list.New(),
		connCallbacks:       connCallbacks,
		streamConnCallbacks: streamConnCallbacks,
	}
}

// types.StreamConnection
func (csw *clientStreamWrapper) Dispatch(buffer types.IoBuffer) {}

func (csw *clientStreamWrapper) Protocol() types.Protocol {
	return protocol.HTTP1
}

func (csw *clientStreamWrapper) GoAway() {
	// todo
}

func (csw *clientStreamWrapper) OnGoAway() {
	csw.streamConnCallbacks.OnGoAway()
}

func (csw *clientStreamWrapper) NewStream(text context.Context, streamID string, responseDecoder types.StreamReceiver) types.StreamSender {
	stream := &clientStream{
		stream: stream{
			context:  context.WithValue(csw.context, types.ContextKeyStreamID, streamID),
			receiver: responseDecoder,
		},
		wrapper: csw,
	}

	csw.asMutex.Lock()
	stream.element = csw.activeStreams.PushBack(stream)
	csw.asMutex.Unlock()

	return stream
}

// types.ServerStreamConnection
type serverStreamConnection struct {
	streamConnection
	serverStreamConnCallbacks types.ServerStreamConnectionEventListener
}

func newServerStreamConnection(context context.Context, connection types.Connection,
	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	ssc := &serverStreamConnection{
		streamConnection: streamConnection{
			context:       context,
			rawConnection: connection.RawConn(),
		},
		serverStreamConnCallbacks: callbacks,
	}

	fasthttp.ServeConn(connection.RawConn(), ssc.ServeHTTP)

	return ssc
}

func (ssc *serverStreamConnection) OnGoAway() {
	ssc.serverStreamConnCallbacks.OnGoAway()
}

//作为PROXY的STREAM SERVER
func (ssc *serverStreamConnection) ServeHTTP(ctx *fasthttp.RequestCtx) {
	//generate stream id using global counter
	streamID := protocol.GenerateIDString()

	s := &serverStream{
		stream: stream{
			context: context.WithValue(ssc.context, types.ContextKeyStreamID, streamID),
		},
		ctx:              ctx,
		connection:       ssc,
		responseDoneChan: make(chan bool, 1),
	}

	s.receiver = ssc.serverStreamConnCallbacks.NewStream(s.stream.context, streamID, s)

	ssc.activeStream = &s.stream

	if atomic.LoadInt32(&s.readDisableCount) <= 0 {
		s.handleRequest()
	}

	select {
	case <-s.responseDoneChan:
		s.ctx = nil
		ssc.activeStream = nil
	}
}

// types.Stream
// types.StreamSender
type stream struct {
	context context.Context

	readDisableCount int32

	receiver  types.StreamReceiver
	streamCbs []types.StreamEventListener
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

	// NOTICE: fasthttp ctx and its member not allowed holding by others after request handle finished
	request  *fasthttp.Request
	response *fasthttp.Response

	element *list.Element
	wrapper *clientStreamWrapper
}

// types.StreamSender
func (s *clientStream) AppendHeaders(headersIn interface{}, endStream bool) error {
	headers, _ := headersIn.(map[string]string)

	if s.request == nil {
		s.request = fasthttp.AcquireRequest()
		s.request.Header.SetMethod(http.MethodGet)
		s.request.SetRequestURI(fmt.Sprintf("http://%s/", s.wrapper.client.Addr))
	}

	if method, ok := headers[types.HeaderMethod]; ok {
		s.request.Header.SetMethod(method)
		delete(headers, types.HeaderMethod)
	}

	if host, ok := headers[types.HeaderHost]; ok {
		s.request.SetHost(host)
		delete(headers, types.HeaderHost)
	}

	if path, ok := headers[protocol.MosnHeaderPathKey]; ok {
		s.request.SetRequestURI(fmt.Sprintf("http://%s%s", s.wrapper.client.Addr, path))
		delete(headers, protocol.MosnHeaderPathKey)
	}

	encodeReqHeader(s.request, headers)

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *clientStream) AppendData(data types.IoBuffer, endStream bool) error {
	if s.request == nil {
		s.request = fasthttp.AcquireRequest()
	}

	s.request.SetBody(data.Bytes())

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *clientStream) AppendTrailers(trailers map[string]string) error {
	s.endStream()

	return nil
}

func (s *clientStream) endStream() {
	go s.doSend()
}

func (s *clientStream) ReadDisable(disable bool) {
	//s.connection.logger.Debugf("high watermark on h2 stream client")

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
	if s.response == nil {
		s.response = fasthttp.AcquireResponse()
	}

	err := s.wrapper.client.Do(s.request, s.response)

	if err != nil {
		log.DefaultLogger.Errorf("http1 client stream send error: %+s", err)
		s.wrapper.connCallbacks.OnEvent(types.RemoteClose)
	} else {

		if atomic.LoadInt32(&s.readDisableCount) <= 0 {
			s.handleResponse()
		}
	}
}

func (s *clientStream) handleResponse() {
	if s.response != nil {
		s.receiver.OnReceiveHeaders(decodeRespHeader(s.response.Header), false)
		buf := buffer.NewIoBufferBytes(s.response.Body())
		s.receiver.OnReceiveData(buf, true)

		s.wrapper.asMutex.Lock()
		s.request = nil
		s.response = nil
		s.wrapper.activeStreams.Remove(s.element)
		s.wrapper.asMutex.Unlock()
	}
}

func (s *clientStream) GetStream() types.Stream {
	return s
}

type serverStream struct {
	stream

	// NOTICE: fasthttp ctx and its member not allowed holding by others after request handle finished
	ctx *fasthttp.RequestCtx

	connection       *serverStreamConnection
	responseDoneChan chan bool
}

// types.StreamSender
func (s *serverStream) AppendHeaders(headerIn interface{}, endStream bool) error {
	headers, _ := headerIn.(map[string]string)

	if status, ok := headers[types.HeaderStatus]; ok {
		statusCode, _ := strconv.Atoi(string(status))
		s.ctx.SetStatusCode(statusCode)
		delete(headers, types.HeaderStatus)
	}

	encodeRespHeader(&s.ctx.Response, headers)

	if endStream {
		s.endStream()
	}
	return nil
}

func (s *serverStream) AppendData(data types.IoBuffer, endStream bool) error {
	s.ctx.SetBody(data.Bytes())

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *serverStream) AppendTrailers(trailers map[string]string) error {
	s.endStream()
	return nil
}

func (s *serverStream) endStream() {
	s.doSend()
	s.responseDoneChan <- true

	s.connection.activeStream = nil
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
	// fasthttp will automatically flush headers and body
}

func (s *serverStream) handleRequest() {
	if s.ctx != nil {

		// header
		header := decodeReqHeader(s.ctx.Request.Header)

		// set host header if not found, just for insurance
		if _, ok := header[protocol.MosnHeaderHostKey]; !ok {
			header[protocol.MosnHeaderHostKey] = string(s.ctx.Host())
		}

		// set path header if not found
		if _, ok := header[protocol.MosnHeaderPathKey]; !ok {
			header[protocol.MosnHeaderPathKey] = string(s.ctx.Path())
		}

		// set query string header if not found
		if _, ok := header[protocol.MosnHeaderQueryStringKey]; !ok {
			header[protocol.MosnHeaderQueryStringKey] = string(s.ctx.URI().QueryString())
		}

		s.receiver.OnReceiveHeaders(header, false)

		// data remove detect
		if s.connection.activeStream != nil {
			buf := buffer.NewIoBufferBytes(s.ctx.Request.Body())
			s.receiver.OnReceiveData(buf, true)
			//no Trailer in Http/1.x
		}
	}
}

func (s *serverStream) GetStream() types.Stream {
	return s
}

func encodeReqHeader(req *fasthttp.Request, in map[string]string) {
	for k, v := range in {
		req.Header.Set(k, v)
	}
}

func encodeRespHeader(resp *fasthttp.Response, in map[string]string) {
	for k, v := range in {
		resp.Header.Set(k, v)
	}
}

func decodeReqHeader(in fasthttp.RequestHeader) (out map[string]string) {
	out = make(map[string]string, in.Len())

	in.VisitAll(func(key, value []byte) {
		// convert to lower case for internal process
		out[strings.ToLower(string(key))] = string(value)
	})

	return
}

func decodeRespHeader(in fasthttp.ResponseHeader) (out map[string]string) {
	out = make(map[string]string, in.Len())

	in.VisitAll(func(key, value []byte) {
		// convert to lower case for internal process
		out[strings.ToLower(string(key))] = string(value)
	})

	// inherit upstream's response status
	out[types.HeaderStatus] = strconv.Itoa(in.StatusCode())

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
