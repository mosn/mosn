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
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/valyala/fasthttp"
	"bufio"
	"errors"
)

var errConnClose = errors.New("connection closed")

func init() {
	str.Register(protocol.HTTP1, &streamConnFactory{})
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

// types.StreamConnection
// types.StreamConnectionEventListener
type streamConnection struct {
	context context.Context

	conn          types.Connection
	connCallbacks types.ConnectionEventListener

	bufChan chan types.IoBuffer
	br      *bufio.Reader
	bw      *bufio.Writer

	logger log.Logger
}

// types.StreamConnection
func (conn *streamConnection) Dispatch(buffer types.IoBuffer) {
	conn.bufChan <- buffer
	<-conn.bufChan
}

func (conn *streamConnection) Protocol() types.Protocol {
	return protocol.HTTP1
}

func (conn *streamConnection) GoAway() {
	// todo
}

func (conn *streamConnection) Read(p []byte) (n int, err error) {
	data, ok := <-conn.bufChan

	// Connection close
	if !ok {
		err = errConnClose
		return
	}

	n = copy(p, data.Bytes())
	data.Drain(n)
	//fmt.Printf("http1 Read : %v\n", p)
	conn.bufChan <- nil
	return
}

func (conn *streamConnection) Write(p []byte) (n int, err error) {
	n = len(p)
	err = conn.conn.Write(buffer.NewIoBufferBytes(p))
	return
}

// conn callbacks
func (conn *streamConnection) OnEvent(event types.ConnectionEvent) {
	if event.IsClose() || event.ConnectFailure() {
		close(conn.bufChan)
	}
}

// types.ClientStreamConnection
type clientStreamConnection struct {
	streamConnection

	stream              *clientStream
	connCallbacks       types.ConnectionEventListener
	streamConnCallbacks types.StreamConnectionEventListener
}

func newClientStreamConnection(context context.Context, connection types.ClientConnection,
	streamConnCallbacks types.StreamConnectionEventListener,
	connCallbacks types.ConnectionEventListener) types.ClientStreamConnection {

	csc := &clientStreamConnection{
		streamConnection: streamConnection{
			context: context,
			conn:    connection,
			bufChan: make(chan types.IoBuffer),
		},
		connCallbacks:       connCallbacks,
		streamConnCallbacks: streamConnCallbacks,
	}

	connection.AddConnectionEventListener(csc)

	csc.br = bufio.NewReader(csc)
	csc.bw = bufio.NewWriter(csc)

	go csc.serve()

	return csc
}

func (csc *clientStreamConnection) serve() {
	for {
		// 1. blocking read using fasthttp.Request.Read
		response := fasthttp.AcquireResponse()
		err := response.Read(csc.br)
		if err != nil {
			if err != errConnClose {
				log.DefaultLogger.Errorf("Http client codec goroutine error: %s", err)
			}
			return
		}

		// 2. response processing
		s := csc.stream
		s.response = response

		if atomic.LoadInt32(&s.readDisableCount) <= 0 {
			s.handleResponse()
		}
	}
}

func (csc *clientStreamConnection) GoAway() {
	// todo
}

func (csc *clientStreamConnection) OnGoAway() {
	csc.streamConnCallbacks.OnGoAway()
}

func (csc *clientStreamConnection) NewStream(ctx context.Context, streamID string, receiver types.StreamReceiver) types.StreamSender {
	s := &clientStream{
		stream: stream{
			context:  context.WithValue(ctx, types.ContextKeyStreamID, streamID),
			receiver: receiver,
		},
		connection: csc,
	}

	csc.stream = s

	return s
}

// types.ServerStreamConnection
type serverStreamConnection struct {
	streamConnection

	stream                    *serverStream
	serverStreamConnCallbacks types.ServerStreamConnectionEventListener
}

func newServerStreamConnection(context context.Context, connection types.Connection,
	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	ssc := &serverStreamConnection{
		streamConnection: streamConnection{
			context: context,
			conn:    connection,
			bufChan: make(chan types.IoBuffer),
		},
		serverStreamConnCallbacks: callbacks,
	}

	ssc.br = bufio.NewReader(ssc)
	ssc.bw = bufio.NewWriter(ssc)

	//fasthttp.ServeConn(connection.RawConn(), ssc.ServeHTTP)
	go ssc.serve()

	return ssc
}

func (ssc *serverStreamConnection) serve() {
	for {
		// 1. blocking read using fasthttp.Request.Read
		request := fasthttp.AcquireRequest()
		err := request.Read(ssc.br)
		if err != nil {
			if err != errConnClose {
				log.DefaultLogger.Errorf("Http server codec goroutine error: %s", err)
			}
			return
		}

		// 2. request processing

		// generate stream id using global counter
		streamID := protocol.GenerateIDString()

		s := &serverStream{
			stream: stream{
				request:  request,
				response: fasthttp.AcquireResponse(),
				context:  context.WithValue(ssc.context, types.ContextKeyStreamID, streamID),
			},
			connection:       ssc,
			responseDoneChan: make(chan bool, 1),
		}

		s.receiver = ssc.serverStreamConnCallbacks.NewStream(s.stream.context, streamID, s)

		ssc.stream = s

		if atomic.LoadInt32(&s.readDisableCount) <= 0 {
			s.handleRequest()
		}

		select {
		case <-s.responseDoneChan:
		}
	}
}

func (ssc *serverStreamConnection) OnGoAway() {
	ssc.serverStreamConnCallbacks.OnGoAway()
}

// types.Stream
// types.StreamSender
type stream struct {
	context context.Context

	readDisableCount int32

	// NOTICE: fasthttp ctx and its member not allowed holding by others after request handle finished
	request  *fasthttp.Request
	response *fasthttp.Response

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

	connection *clientStreamConnection
}

// types.StreamSender
func (s *clientStream) AppendHeaders(context context.Context, headersIn types.HeaderMap, endStream bool) error {
	headers, _ := headersIn.(protocol.CommonHeader)

	if s.request == nil {
		s.request = fasthttp.AcquireRequest()
		// TODO: protocol convert in pkg/protocol
		// f the request contains body, use "POST" as default, the http request method will be setted by MosnHeaderMethod
		if endStream {
			s.request.Header.SetMethod(http.MethodGet)
		} else {
			s.request.Header.SetMethod(http.MethodPost)
		}
		s.request.SetRequestURI(fmt.Sprintf("http://%s/", s.connection.conn.RemoteAddr()))
	}

	if method, ok := headers[protocol.MosnHeaderMethod]; ok {
		s.request.Header.SetMethod(method)
		delete(headers, protocol.MosnHeaderMethod)
	}

	if host, ok := headers[protocol.MosnHeaderHostKey]; ok {
		s.request.SetHost(host)
		delete(headers, protocol.MosnHeaderHostKey)
	}

	var URI string

	if path, ok := headers[protocol.MosnHeaderPathKey]; ok {
		URI = fmt.Sprintf("http://%s%s", s.connection.conn.RemoteAddr(), path)
		delete(headers, protocol.MosnHeaderPathKey)
	}

	if URI != "" {

		if queryString, ok := headers[protocol.MosnHeaderQueryStringKey]; ok {
			URI += "?" + queryString
		}

		s.request.SetRequestURI(URI)
	}

	if _, ok := headers[protocol.MosnHeaderQueryStringKey]; ok {
		delete(headers, protocol.MosnHeaderQueryStringKey)

	}

	encodeReqHeader(s.request, headers)

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *clientStream) AppendData(context context.Context, data types.IoBuffer, endStream bool) error {
	if s.request == nil {
		s.request = fasthttp.AcquireRequest()
	}

	s.request.SetBody(data.Bytes())

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *clientStream) AppendTrailers(context context.Context, trailers types.HeaderMap) error {
	s.endStream()

	return nil
}

func (s *clientStream) endStream() {
	s.doSend()
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
	if _, err := s.request.WriteTo(s.connection); err != nil {
		log.DefaultLogger.Errorf("http1 client stream send error: %+s", err)

		//TODO
		s.connection.connCallbacks.OnEvent(types.RemoteClose)
	}
}

func (s *clientStream) handleResponse() {
	if s.response != nil {
		s.receiver.OnReceiveHeaders(s.context, protocol.CommonHeader(decodeRespHeader(s.response.Header)), false)
		buf := buffer.NewIoBufferBytes(s.response.Body())
		s.receiver.OnReceiveData(s.context, buf, true)

		//TODO cannot recycle immediately, headers might be used by proxy logic
		s.request = nil
		s.response = nil
	}
}

func (s *clientStream) GetStream() types.Stream {
	return s
}

type serverStream struct {
	stream

	connection       *serverStreamConnection
	responseDoneChan chan bool
}

// types.StreamSender
func (s *serverStream) AppendHeaders(context context.Context, headerIn types.HeaderMap, endStream bool) error {
	headers, _ := headerIn.(protocol.CommonHeader)

	if status, ok := headers[types.HeaderStatus]; ok {
		statusCode, _ := strconv.Atoi(string(status))
		s.response.SetStatusCode(statusCode)
		delete(headers, types.HeaderStatus)
	}

	encodeRespHeader(s.response, headers)

	if endStream {
		s.endStream()
	}
	return nil
}

func (s *serverStream) AppendData(context context.Context, data types.IoBuffer, endStream bool) error {
	s.response.SetBody(data.Bytes())

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *serverStream) AppendTrailers(context context.Context, trailers types.HeaderMap) error {
	s.endStream()
	return nil
}

func (s *serverStream) endStream() {
	s.doSend()
	s.responseDoneChan <- true

	// clean up & recycle
	s.connection.stream = nil
	if s.request != nil {
		fasthttp.ReleaseRequest(s.request)
	}
	if s.response != nil {
		fasthttp.ReleaseResponse(s.response)
	}
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
	s.response.WriteTo(s.connection)
}

func (s *serverStream) handleRequest() {
	if s.request != nil {

		// header
		header := decodeReqHeader(s.request.Header)

		// set host header if not found, just for insurance
		if _, ok := header[protocol.MosnHeaderHostKey]; !ok {
			header[protocol.MosnHeaderHostKey] = string(s.request.Host())
		}

		// todo: this is a hack for set ":authority" header if not found
		if _, ok := header[protocol.IstioHeaderHostKey]; !ok {
			header[protocol.IstioHeaderHostKey] = string(s.request.Host())
		}

		// set path header if not found
		if _, ok := header[protocol.MosnHeaderPathKey]; !ok {
			header[protocol.MosnHeaderPathKey] = string(s.request.URI().Path())
		}

		// set query string header if not found
		if _, ok := header[protocol.MosnHeaderQueryStringKey]; !ok {
			header[protocol.MosnHeaderQueryStringKey] = string(s.request.URI().QueryString())
		}

		// set method string header if not found
		if _, ok := header[protocol.MosnHeaderMethod]; !ok {
			header[protocol.MosnHeaderMethod] = string(s.request.Header.Method())
		}

		s.receiver.OnReceiveHeaders(s.context, protocol.CommonHeader(header), false)

		// data remove detect
		if s.connection.stream != nil {
			s.receiver.OnReceiveData(s.context, buffer.NewIoBufferBytes(s.request.Body()), true)
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
