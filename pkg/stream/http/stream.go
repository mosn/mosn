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
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"mosn.io/mosn/pkg/variable"

	"github.com/valyala/fasthttp"
	"mosn.io/api"
	mbuffer "mosn.io/mosn/pkg/buffer"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	mosnhttp "mosn.io/mosn/pkg/protocol/http"
	str "mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"
)

func init() {
	str.Register(protocol.HTTP1, &streamConnFactory{})
	protocol.RegisterProtocolConfigHandler(protocol.HTTP1, streamConfigHandler)
}

const defaultMaxRequestBodySize = 4 * 1024 * 1024
const defaultMaxHeaderSize = 8 * 1024

var (
	errConnClose = errors.New("connection closed")

	strResponseContinue = []byte("HTTP/1.1 100 Continue\r\n\r\n")
	strErrorResponse    = []byte("HTTP/1.1 400 Bad Request\r\n\r\n")

	HKConnection = []byte("Connection") // header key 'Connection'
	HVKeepAlive  = []byte("keep-alive") // header value 'keep-alive'

	minMethodLengh = len("GET")
	maxMethodLengh = len("CONNECT")
	httpMethod     = map[string]struct{}{
		"OPTIONS": {},
		"GET":     {},
		"HEAD":    {},
		"POST":    {},
		"PUT":     {},
		"DELETE":  {},
		"TRACE":   {},
		"CONNECT": {},
		// See https://datatracker.ietf.org/doc/html/rfc2068#section-19.6.1
		"PATCH":  {},
		"LINK":   {},
		"UNLINK": {},
	}
)

type streamConnFactory struct{}

func (f *streamConnFactory) CreateClientStream(context context.Context, connection types.ClientConnection,
	streamConnCallbacks types.StreamConnectionEventListener, connCallbacks api.ConnectionEventListener) types.ClientStreamConnection {
	return newClientStreamConnection(context, connection, streamConnCallbacks, connCallbacks)
}

func (f *streamConnFactory) CreateServerStream(context context.Context, connection api.Connection,
	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	return newServerStreamConnection(context, connection, callbacks)
}

func (f *streamConnFactory) CreateBiDirectStream(context context.Context, connection types.ClientConnection,
	clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {
	return nil
}

func (f *streamConnFactory) ProtocolMatch(context context.Context, prot string, magic []byte) error {
	if len(magic) < minMethodLengh {
		return str.EAGAIN
	}
	size := len(magic)
	if len(magic) > maxMethodLengh {
		size = maxMethodLengh
	}

	for i := minMethodLengh; i <= size; i++ {
		if _, ok := httpMethod[string(magic[0:i])]; ok {
			return nil
		}
	}

	if size < maxMethodLengh {
		return str.EAGAIN
	} else {
		return str.FAILED
	}
}

// types.StreamConnection
// types.StreamConnectionEventListener
type streamConnection struct {
	context context.Context

	conn              api.Connection
	connEventListener api.ConnectionEventListener
	resetReason       types.StreamResetReason

	bufChan    chan buffer.IoBuffer
	endRead    chan struct{}
	connClosed chan bool

	br *bufio.Reader
}

// types.StreamConnection
func (sc *streamConnection) Dispatch(buffer buffer.IoBuffer) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("[stream] [http] connection has closed. Connection = %d, Local Address = %+v, Remote Address = %+v, err = %+v",
				sc.conn.ID(), sc.conn.LocalAddr(), sc.conn.RemoteAddr(), r)
		}
	}()

	for buffer.Len() > 0 {
		sc.bufChan <- buffer
		<-sc.endRead
	}
}

func (conn *streamConnection) Protocol() types.ProtocolName {
	return protocol.HTTP1
}

func (conn *streamConnection) EnableWorkerPool() bool {
	return true
}

func (conn *streamConnection) GoAway() {}

func (conn *streamConnection) Read(p []byte) (n int, err error) {
	// conn.bufChan <- nil Maybe caused panic error when connection closed.
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Infof("[stream] [http] connection has closed. Connection = %d, Local Address = %+v, Remote Address = %+v, err = %+v",
				conn.conn.ID(), conn.conn.LocalAddr(), conn.conn.RemoteAddr(), r)
		}
	}()

	data, ok := <-conn.bufChan

	// Connection close
	if !ok {
		// Compatible with the fasthttp,
		// it should be set err = IO.EOF when the peer connection is closed.
		switch conn.resetReason {
		case types.UpstreamReset:
			err = io.EOF
		default:
			err = errConnClose
		}

		return
	}

	n = copy(p, data.Bytes())
	data.Drain(n)
	conn.endRead <- struct{}{}
	return
}

func (conn *streamConnection) Write(p []byte) (n int, err error) {
	n = len(p)

	// TODO avoid copy
	buf := buffer.GetIoBuffer(n)
	buf.Write(p)

	err = conn.conn.Write(buf)
	return
}

func (conn *streamConnection) Reset(reason types.StreamResetReason) {
	// We need to set 'conn.resetReason' before 'close(conn.bufChan)'
	// because streamConnection's Read will do some processing depends it.
	conn.resetReason = reason
	close(conn.bufChan)
	close(conn.endRead)
	close(conn.connClosed)
}

// types.ClientStreamConnection
type clientStreamConnection struct {
	streamConnection

	stream                        *clientStream
	requestSent                   chan bool
	mutex                         sync.RWMutex
	connectionEventListener       api.ConnectionEventListener
	streamConnectionEventListener types.StreamConnectionEventListener
}

func newClientStreamConnection(ctx context.Context, connection types.ClientConnection,
	streamConnCallbacks types.StreamConnectionEventListener,
	connCallbacks api.ConnectionEventListener) types.ClientStreamConnection {

	csc := &clientStreamConnection{
		streamConnection: streamConnection{
			context:    ctx,
			conn:       connection,
			bufChan:    make(chan buffer.IoBuffer),
			endRead:    make(chan struct{}),
			connClosed: make(chan bool, 1),
		},
		connectionEventListener:       connCallbacks,
		streamConnectionEventListener: streamConnCallbacks,
		requestSent:                   make(chan bool, 1),
	}

	// Per-connection buffer size for responses' reading.
	// This also limits the maximum header size, default 8192.
	maxResponseHeaderSize := 0
	if pgc := mosnctx.Get(ctx, types.ContextKeyProxyGeneralConfig); pgc != nil {
		if extendConfig, ok := pgc.(map[string]interface{}); ok {
			if v, ok := extendConfig["max_header_size"]; ok {
				// json.Unmarshal stores float64 for JSON numbers in the interface{}
				// see doc: https://golang.org/pkg/encoding/json/#Unmarshal
				if fv, ok := v.(float64); ok {
					maxResponseHeaderSize = int(fv)
				}
			}
		}
	}
	if maxResponseHeaderSize <= 0 {
		maxResponseHeaderSize = defaultMaxHeaderSize
	}

	csc.br = bufio.NewReaderSize(csc, maxResponseHeaderSize)

	utils.GoWithRecover(func() {
		csc.serve()
	}, nil)

	return csc
}

func (conn *clientStreamConnection) serve() {
	for {
		select {
		case <-conn.requestSent:
		case <-conn.connClosed:
			return
		}

		s := conn.stream
		buffers := httpBuffersByContext(s.ctx)
		s.response = &buffers.clientResponse
		request := &buffers.serverRequest

		// Response.Read() skips reading body if set to true.
		// Use it for reading HEAD responses.
		if request.Header.IsHead() {
			s.response.SkipBody = true
		}

		// 1. blocking read using fasthttp.Response.Read
		err := s.response.Read(conn.br)
		if err != nil {
			if s != nil {
				log.Proxy.Errorf(s.connection.context, "[stream] [http] client stream connection wait response error: %s", err)
				reason := conn.resetReason
				if reason == "" {
					reason = types.StreamRemoteReset
				}
				s.ResetStream(reason)
			}
			return
		}

		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(s.stream.ctx, "[stream] [http] receive response, requestId = %v", s.stream.id)
		}

		// 2. response processing
		resetConn := false
		if s.response.ConnectionClose() {
			resetConn = true
		}

		// 3. local reset if header 'Connection: close' exists
		if resetConn {
			// goaway the connpool
			s.connection.streamConnectionEventListener.OnGoAway()
		}

		if atomic.LoadInt32(&s.readDisableCount) <= 0 {
			s.handleResponse()
		}
	}
}

func (conn *clientStreamConnection) GoAway() {}

func (conn *clientStreamConnection) NewStream(ctx context.Context, receiver types.StreamReceiveListener) types.StreamSender {
	id := protocol.GenerateID()
	buffers := httpBuffersByContext(ctx)
	s := &buffers.clientStream
	s.stream = stream{
		id:       id,
		ctx:      mosnctx.WithValue(ctx, types.ContextKeyStreamID, id),
		request:  &buffers.clientRequest,
		receiver: receiver,
	}
	s.connection = conn

	conn.mutex.Lock()
	conn.stream = s
	conn.mutex.Unlock()
	return s
}

func (conn *clientStreamConnection) ActiveStreamsNum() int {
	conn.mutex.RLock()
	defer conn.mutex.RUnlock()

	if conn.stream == nil {
		return 0
	} else {
		return 1
	}
}

func (conn *clientStreamConnection) CheckReasonError(connected bool, event api.ConnectionEvent) (types.StreamResetReason, bool) {
	reason := types.StreamConnectionSuccessed
	if event.IsClose() || event.ConnectFailure() {
		reason = types.StreamConnectionFailed
		if connected {
			switch event {
			case api.RemoteClose:
				reason = types.UpstreamReset
			default:
				reason = types.StreamConnectionTermination
			}
		}
		return reason, false

	}
	return reason, true
}

type StreamConfig struct {
	MaxHeaderSize      int `json:"max_header_size,omitempty"`
	MaxRequestBodySize int `json:"max_request_body_size,omitempty"`
}

var defaultStreamConfig = StreamConfig{
	// Per-connection buffer size for requests' reading.
	// This also limits the maximum header size, default 8192.
	MaxHeaderSize: defaultMaxHeaderSize,
	// 0 is means no limit request body size
	MaxRequestBodySize: 0,
}

func streamConfigHandler(v interface{}) interface{} {
	extendConfig, ok := v.(map[string]interface{})
	if !ok {
		return defaultStreamConfig
	}

	// extract http1 config
	config, ok := extendConfig[string(protocol.HTTP1)]
	if !ok {
		config = extendConfig
	}
	configBytes, err := json.Marshal(config)
	if err != nil {
		return defaultStreamConfig
	}
	streamConfig := defaultStreamConfig
	if err := json.Unmarshal(configBytes, &streamConfig); err != nil {
		return defaultStreamConfig
	}
	return streamConfig

}

func parseStreamConfig(ctx context.Context) StreamConfig {
	streamConfig := defaultStreamConfig
	// get extend config from ctx
	pgc := mosnctx.Get(ctx, types.ContextKeyProxyGeneralConfig)
	if cfg, ok := pgc.(StreamConfig); ok {
		streamConfig = cfg
	}
	return streamConfig
}

// types.ServerStreamConnection
type serverStreamConnection struct {
	streamConnection
	contextManager *str.ContextManager
	config         StreamConfig

	close bool

	stream                   *serverStream
	mutex                    sync.RWMutex
	serverStreamConnListener types.ServerStreamConnectionEventListener
}

func newServerStreamConnection(ctx context.Context, connection api.Connection,
	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	ssc := &serverStreamConnection{
		streamConnection: streamConnection{
			context:    ctx,
			conn:       connection,
			bufChan:    make(chan buffer.IoBuffer),
			endRead:    make(chan struct{}),
			connClosed: make(chan bool, 1),
		},
		config:                   parseStreamConfig(ctx),
		contextManager:           str.NewContextManager(ctx),
		serverStreamConnListener: callbacks,
	}

	// init first context
	ssc.contextManager.Next()

	ssc.br = bufio.NewReaderSize(ssc, ssc.config.MaxHeaderSize)

	// Reset would not be called in server-side scene, so add listener for connection event
	connection.AddConnectionEventListener(ssc)
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(ctx, "new http server stream connection, stream config: %v", ssc.config)
	}

	// set not support transfer connection
	ssc.conn.SetTransferEventListener(func() bool {
		ssc.close = true
		return false
	})

	utils.GoWithRecover(func() {
		ssc.serve()
	}, nil)

	return ssc
}

func (conn *serverStreamConnection) OnEvent(event api.ConnectionEvent) {
	if event.IsClose() {
		var reason types.StreamResetReason
		switch event {
		case api.RemoteClose:
			reason = types.UpstreamReset
		default:
			reason = types.StreamConnectionTermination
		}
		conn.Reset(reason)
	}
}

func (conn *serverStreamConnection) serve() {
	for {
		// 1. pre alloc stream-level ctx with bufferCtx
		ctx := conn.contextManager.Get()
		buffers := httpBuffersByContext(ctx)
		request := &buffers.serverRequest

		// 0 is means no limit request body size
		maxRequestBodySize := conn.config.MaxRequestBodySize

		// 2. blocking read using fasthttp.Request.Read
		err := request.ReadLimitBody(conn.br, maxRequestBodySize)
		if err == nil {
			// 3. 'Expect: 100-continue' request handling.
			// See http://www.w3.org/Protocols/rfc2616/rfc2616-sec8.html for details.
			if request.MayContinue() {
				// Send 'HTTP/1.1 100 Continue' response.
				conn.conn.Write(buffer.NewIoBufferBytes(strResponseContinue))

				// read request body
				err = request.ContinueReadBody(conn.br, maxRequestBodySize)

				// remove 'Expect' header, so it would not be sent to the upstream
				request.Header.Del("Expect")
			}
		}
		if err != nil {
			// ErrNothingRead is returned that means just a keep-alive connection
			// closing down either because the remote closed it or because
			// or a read timeout on our side. Either way just close the connection
			// and don't return any error response.
			// Refer https://github.com/valyala/fasthttp/commit/598a52272abafde3c5bebd7cc1972d3bead7a1f7
			_, errNothingRead := err.(fasthttp.ErrNothingRead)
			if err != errConnClose && err != io.EOF && !errNothingRead {
				// write error response
				conn.conn.Write(buffer.NewIoBufferBytes(strErrorResponse))

				// close connection with flush
				conn.conn.Close(api.FlushWrite, api.LocalClose)
			}
			return
		}

		id := protocol.GenerateID()
		s := &buffers.serverStream

		// 4. request processing
		s.stream = stream{
			id:       id,
			ctx:      mosnctx.WithValue(ctx, types.ContextKeyStreamID, id),
			request:  request,
			response: &buffers.serverResponse,
		}
		s.connection = conn
		s.responseDoneChan = make(chan bool, 1)
		s.header = mosnhttp.RequestHeader{&s.request.Header}

		// save the connection state, because clientStream AppendHeaders will modify it.
		if s.header.ConnectionClose() {
			s.connection.close = true
		}

		var span api.Span
		if trace.IsEnabled() {
			tracer := trace.Tracer(protocol.HTTP1)
			if tracer != nil {
				span = tracer.Start(ctx, s.header, time.Now())
			}
		}
		s.stream.ctx = s.connection.contextManager.InjectTrace(ctx, span)

		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(s.stream.ctx, "[stream] [http] new stream detect, requestId = %v", s.stream.id)
		}

		s.receiver = conn.serverStreamConnListener.NewStreamDetect(s.stream.ctx, s, span)

		conn.mutex.Lock()
		conn.stream = s
		conn.mutex.Unlock()

		if atomic.LoadInt32(&s.readDisableCount) <= 0 {
			s.handleRequest(s.stream.ctx)
		}

		// 5. wait for proxy done
		select {
		case <-s.responseDoneChan:
		case <-conn.connClosed:
			return
		}

		conn.contextManager.Next()
	}
}

func (conn *serverStreamConnection) ActiveStreamsNum() int {
	conn.mutex.RLock()
	defer conn.mutex.RUnlock()

	if conn.stream == nil {
		return 0
	} else {
		return 1
	}
}

func (conn *serverStreamConnection) CheckReasonError(connected bool, event api.ConnectionEvent) (types.StreamResetReason, bool) {
	reason := types.StreamConnectionSuccessed
	if event.IsClose() || event.ConnectFailure() {
		reason = types.StreamConnectionFailed
		if connected {
			switch event {
			case api.RemoteClose:
				reason = types.UpstreamReset
			default:
				reason = types.StreamConnectionTermination
			}
		}
		return reason, false

	}
	return reason, true
}

// types.Stream
// types.StreamSender
type stream struct {
	str.BaseStream

	id               uint64
	readDisableCount int32
	ctx              context.Context

	// NOTICE: fasthttp ctx and its member not allowed holding by others after request handle finished
	request  *fasthttp.Request
	response *fasthttp.Response

	receiver types.StreamReceiveListener
}

// types.Stream
func (s *stream) ID() uint64 {
	return s.id
}

// types.StreamSender for request
type clientStream struct {
	stream

	connection *clientStreamConnection
}

// types.StreamSender
func (s *clientStream) AppendHeaders(context context.Context, headersIn types.HeaderMap, endStream bool) error {
	headers := headersIn.(mosnhttp.RequestHeader)

	// TODO: protocol convert in pkg/protocol
	//if the request contains body, use "POST" as default, the http request method will be setted by MosnHeaderMethod
	if endStream {
		headers.SetMethod(http.MethodGet)
	} else {
		headers.SetMethod(http.MethodPost)
	}

	// clear 'Connection:close' header for keepalive connection with upstream
	if headers.ConnectionClose() {
		headers.Del("Connection")
	}

	FillRequestHeadersFromCtxVar(context, headers, s.connection.conn.RemoteAddr())

	// copy headers
	headers.CopyTo(&s.request.Header)

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *clientStream) AppendData(context context.Context, data buffer.IoBuffer, endStream bool) error {
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
	err := s.doSend()

	if err != nil {
		log.Proxy.Errorf(s.stream.ctx, "[stream] [http] send client request error: %+v", err)

		if err == types.ErrConnectionHasClosed {
			s.ResetStream(types.StreamConnectionFailed)
		} else {
			s.ResetStream(types.StreamLocalReset)
		}
		return
	}

	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(s.stream.ctx, "[stream] [http] send client request, requestId = %v", s.stream.id)
	}
	s.connection.requestSent <- true
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

func (s *clientStream) doSend() (err error) {
	_, err = s.request.WriteTo(s.connection)
	return
}

func (s *clientStream) handleResponse() {
	if s.response != nil {
		header := mosnhttp.ResponseHeader{&s.response.Header}

		statusCode := header.StatusCode()
		status := strconv.Itoa(statusCode)
		// inherit upstream's response status
		variable.SetString(s.ctx, types.VarHeaderStatus, status)

		hasData := true
		if len(s.response.Body()) == 0 {
			hasData = false
		}

		s.connection.mutex.Lock()
		s.connection.stream = nil
		s.connection.mutex.Unlock()

		if s.receiver != nil {
			if hasData {
				s.receiver.OnReceive(s.ctx, header, buffer.NewIoBufferBytes(s.response.Body()), nil)
			} else {
				s.receiver.OnReceive(s.ctx, header, nil, nil)
			}
		}
	}
}

func (s *clientStream) GetStream() types.Stream {
	return s
}

// types.StreamSender for response
type serverStream struct {
	stream

	header           mosnhttp.RequestHeader
	connection       *serverStreamConnection
	responseDoneChan chan bool
}

// types.StreamSender
func (s *serverStream) AppendHeaders(context context.Context, headersIn types.HeaderMap, endStream bool) error {
	switch headers := headersIn.(type) {
	case mosnhttp.RequestHeader:
		// hijack scene
		status, err := variable.GetString(context, types.VarHeaderStatus)
		if err == nil && status != "" {
			statusCode, err := strconv.Atoi(status)
			if err != nil {
				return err
			}
			s.response.SetStatusCode(statusCode)

			FillRequestHeadersFromCtxVar(context, headers, s.connection.conn.RemoteAddr())

			// need to echo all request headers for protocol convert
			headers.VisitAll(func(key, value []byte) {
				s.response.Header.SetBytesKV(key, value)
			})

		}

	case mosnhttp.ResponseHeader:

		status, err := variable.GetString(context, types.VarHeaderStatus)
		if err == nil && status != "" {
			statusCode, _ := strconv.Atoi(status)
			headers.SetStatusCode(statusCode)
		}

		headers.CopyTo(&s.response.Header)
	}

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *serverStream) AppendData(context context.Context, data buffer.IoBuffer, endStream bool) error {
	// SetBodyRaw sets response body and could avoid copying it
	// note: When it's actually sent to the network, it will copy the data once in Write func.
	s.response.SetBodyRaw(data.Bytes())

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
	resetConn := false

	// Response.Write() skips writing body if set to true.
	// Use it for writing HEAD responses.
	if s.request.Header.IsHead() {
		s.response.SkipBody = true
	}

	// check if we need close connection
	if s.connection.close || s.request.Header.ConnectionClose() {
		// should delete 'Connection:keepalive' header
		if !s.response.ConnectionClose() {
			s.response.Header.Del("Connection")
			s.response.SetConnectionClose()
		}
		resetConn = true
	} else if !s.request.Header.IsHTTP11() {
		// Set 'Connection: keep-alive' response header for non-HTTP/1.1 request.
		// There is no need in setting this header for http/1.1, since in http/1.1
		// connections are keep-alive by default.
		s.response.Header.SetCanonical(HKConnection, HVKeepAlive)
	}
	defer s.DestroyStream()

	s.doSend()
	s.responseDoneChan <- true

	if resetConn {
		// close connection
		s.connection.conn.Close(api.FlushWrite, api.LocalClose)
	}

	// clean up & recycle
	s.connection.mutex.Lock()
	s.connection.stream = nil
	s.connection.mutex.Unlock()
}

func (s *serverStream) ReadDisable(disable bool) {
	if disable {
		atomic.AddInt32(&s.readDisableCount, 1)
	} else {
		newCount := atomic.AddInt32(&s.readDisableCount, -1)

		if newCount <= 0 {
			s.handleRequest(s.ctx)
		}
	}
}

func (s *serverStream) doSend() {

	if _, err := s.response.WriteTo(s.connection); err != nil {
		log.Proxy.Errorf(s.stream.ctx, "[stream] [http] send server response error: %+v", err)
	} else {
		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(s.stream.ctx, "[stream] [http] send server response, requestId = %v", s.stream.id)
		}
	}
}

func (s *serverStream) handleRequest(ctx context.Context) {
	if s.request != nil {
		// set non-header info in request-line, like method, uri
		injectCtxVarFromProtocolHeaders(ctx, s.header, s.request.URI())
		hasData := true
		if len(s.request.Body()) == 0 {
			hasData = false
		}

		if hasData {
			s.receiver.OnReceive(s.ctx, s.header, buffer.NewIoBufferBytes(s.request.Body()), nil)
		} else {
			s.receiver.OnReceive(s.ctx, s.header, nil, nil)
		}
	}
}

func (s *serverStream) GetStream() types.Stream {
	return s
}

// consider host, method, path are necessary, but check querystring
func injectCtxVarFromProtocolHeaders(ctx context.Context, header mosnhttp.RequestHeader, uri *fasthttp.URI) {
	// 1. host
	variable.SetString(ctx, types.VarHost, string(uri.Host()))
	// 2. authority
	variable.SetString(ctx, types.VarIstioHeaderHost, string(uri.Host()))

	// 3. method
	variable.SetString(ctx, types.VarMethod, string(header.Method()))

	// 4. path
	variable.SetString(ctx, types.VarPath, string(uri.Path()))

	// 5. path original
	variable.SetString(ctx, types.VarPathOriginal, string(uri.PathOriginal()))

	// 6. querystring
	qs := uri.QueryString()
	if len(qs) > 0 {
		variable.SetString(ctx, types.VarQueryString, string(qs))
	}
}

func buildUrlFromCtxVar(ctx context.Context) string {
	path, _ := variable.GetString(ctx, types.VarPath)
	pathOriginal, _ := variable.GetString(ctx, types.VarPathOriginal)
	queryString, _ := variable.GetString(ctx, types.VarQueryString)

	// different from url.RequestURI() since we should by-pass the original path to upstream
	// even if the original path contains literal space
	var res string

	unescapedPath, err := url.PathUnescape(pathOriginal)
	// path and pathOriginal came from fasthttp.URI that will unescape path and remove duplicate slashes
	// so that /foo/%2Fbar will be /foo/bar(path), but unescapedPath will be /foo//bar
	// We need try fasthttp.URI.SetPath and compare them again.
	// TODO: This condition can be removed if fasthttp fix unescape bug later.
	// See https://github.com/valyala/fasthttp/issues/1030
	if (err == nil && path == unescapedPath) || path == string(fasthttpPath(pathOriginal)) {
		res = pathOriginal
	} else {
		if path == "*" {
			res = "*"
		} else {
			u := url.URL{Path: path}
			res = u.RequestURI()
		}
	}

	if res == "" {
		res = "/"
	}

	if queryString != "" {
		res += "?" + queryString
	}

	return res
}

func FillRequestHeadersFromCtxVar(ctx context.Context, headers mosnhttp.RequestHeader, remoteAddr net.Addr) {
	headers.SetRequestURI(buildUrlFromCtxVar(ctx))

	method, err := variable.GetString(ctx, types.VarMethod)
	if err == nil && method != "" {
		headers.SetMethod(method)
	}

	host, err := variable.GetString(ctx, types.VarHost)
	if err == nil && host != "" {
		headers.SetHost(host)
	} else {
		headers.SetHost(remoteAddr.String())
	}

	host, err = variable.GetString(ctx, types.VarIstioHeaderHost)
	if err == nil && host != "" {
		headers.SetHost(host)
	}

}

// contextManager
type contextManager struct {
	base context.Context
	curr context.Context
}

func (cm *contextManager) next() {
	// new stream-level context based on connection-level's
	cm.curr = mbuffer.NewBufferPoolContext(mosnctx.Clone(cm.base))
}

func fasthttpPath(pathOriginal string) []byte {
	u := &fasthttp.URI{}
	u.SetPath(pathOriginal)
	return u.Path()
}
