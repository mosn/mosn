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

package mhttp2

import (
	"context"
	"sync"

	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/module/http2"
	"github.com/alipay/sofa-mosn/pkg/mtls"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/mhttp2"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	str.Register(protocol.MHTTP2, &streamConnFactory{})
}

type streamConnFactory struct{}

func (f *streamConnFactory) CreateClientStream(context context.Context, connection types.ClientConnection,
	clientCallbacks types.StreamConnectionEventListener, connCallbacks types.ConnectionEventListener) types.ClientStreamConnection {
	return newClientStreamConnection(context, connection, clientCallbacks)
}

func (f *streamConnFactory) CreateServerStream(context context.Context, connection types.Connection,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	return newServerStreamConnection(context, connection, serverCallbacks)
}

func (f *streamConnFactory) CreateBiDirectStream(context context.Context, connection types.ClientConnection,
	clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {
	return nil
}

// types.DecodeFilter
// types.StreamConnection
// types.ClientStreamConnection
// types.ServerStreamConnection
type streamConnection struct {
	ctx    context.Context
	conn   types.Connection
	cm     contextManager
	logger log.Logger

	codecEngine types.ProtocolEngine
}

func (conn *streamConnection) Protocol() types.Protocol {
	return protocol.MHTTP2
}

func (conn *streamConnection) GoAway() {
	// todo
}

// types.Stream
// types.StreamSender
type stream struct {
	ctx       context.Context
	receiver  types.StreamReceiver
	streamCbs []types.StreamEventListener

	id       uint32
	sendData []types.IoBuffer
	conn     types.Connection
}

// ~~ types.Stream
func (s *stream) AddEventListener(cb types.StreamEventListener) {
	s.streamCbs = append(s.streamCbs, cb)
}

func (s *stream) RemoveEventListener(cb types.StreamEventListener) {
	cbIdx := -1

	for i, streamCb := range s.streamCbs {
		if streamCb == cb {
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

func (s *stream) ReadDisable(disable bool) {
	s.conn.SetReadDisable(disable)
}

func (s *stream) BufferLimit() uint32 {
	return s.conn.BufferLimit()
}

func (s *stream) GetStream() types.Stream {
	return s
}

func (s *stream) ID() uint64 {
	return uint64(s.id)
}

func (s *stream) BuildData() types.IoBuffer {
	if s.sendData == nil {
		return buffer.NewIoBuffer(0)
	} else if len(s.sendData) == 1 {
		return s.sendData[0]
	} else {
		size := 0
		for _, buf := range s.sendData {
			size += buf.Len()
		}
		data := buffer.NewIoBuffer(size)
		for _, buf := range s.sendData {
			data.Write(buf.Bytes())
			buffer.PutIoBuffer(buf)
		}
		return data
	}
}

// contextManager
type contextManager struct {
	base context.Context

	curr context.Context
}

func (cm *contextManager) next() {
	// new context
	cm.curr = buffer.NewBufferPoolContext(cm.base)
}

type serverStreamConnection struct {
	streamConnection
	slock   sync.Mutex
	streams map[uint32]*serverStream
	sc      *http2.MServerConn

	serverCallbacks types.ServerStreamConnectionEventListener
}

func newServerStreamConnection(ctx context.Context, connection types.Connection, serverCallbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {

	h2sc := http2.NewServerConn(connection)

	sc := &serverStreamConnection{
		streamConnection: streamConnection{
			ctx:         ctx,
			conn:        connection,
			codecEngine: mhttp2.EngineServer(h2sc),

			cm: contextManager{base: ctx},

			logger: log.ByContext(ctx),
		},
		sc: h2sc,

		serverCallbacks: serverCallbacks,
	}

	// init first context
	sc.cm.next()

	sc.streams = make(map[uint32]*serverStream, 32)

	return sc
}

// types.StreamConnectionM
func (conn *serverStreamConnection) Dispatch(buf types.IoBuffer) {
	for {
		// 1. pre alloc stream-level ctx with bufferCtx
		ctx := conn.cm.curr

		// 2. decode process
		frame, err := conn.codecEngine.Decode(ctx, buf)
		// No enough data
		if err == http2.ErrAGAIN {
			break
		}

		// Do handle staff. Error would also be passed to this function.
		conn.handleFrame(ctx, frame, err)
		if err != nil {
			break
		}

		conn.cm.next()
	}
}

func (conn *serverStreamConnection) handleFrame(ctx context.Context, i interface{}, err error) {
	f, _ := i.(http2.Frame)
	if err != nil {
		conn.handleError(ctx, f, err)
		return
	}
	var h2s *http2.MStream
	var endStream, hasTrailer bool
	var data []byte

	h2s, data, hasTrailer, endStream, err = conn.sc.HandleFrame(ctx, f)

	if err != nil {
		conn.handleError(ctx, f, err)
		return
	}

	if h2s == nil && data == nil && !hasTrailer && !endStream {
		return
	}

	id := f.Header().StreamID

	// header
	if h2s != nil {
		stream, err := conn.onNewStreamDetect(ctx, h2s, endStream)
		if err != nil {
			conn.handleError(ctx, f, err)
			return
		}

		header := decodeHeader(h2s.Request.Header)

		if _, ok := header[types.HeaderMethod]; !ok {
			header[types.HeaderMethod] = h2s.Request.Method
		}

		if _, ok := header[types.HeaderHost]; !ok {
			header[types.HeaderHost] = h2s.Request.Host
		}

		if _, ok := header[types.HeaderPath]; !ok {
			header[types.HeaderPath] = h2s.Request.RequestURI
		}

		if _, ok := header[types.HeaderStreamID]; !ok {
			header[types.HeaderStreamID] = strconv.Itoa(int(id))
		}

		conn.logger.Debugf("http2 server header: %d, %v", id, header)

		stream.receiver.OnReceiveHeaders(ctx, protocol.CommonHeader(header), endStream)
		return
	}

	stream := conn.onStreamRecv(ctx, id, endStream)
	if stream == nil {
		conn.logger.Errorf("http2 server OnStreamRecv error, invaild id = %d", id)
		return
	}

	// data
	if data != nil {
		conn.logger.Debugf("http2 server receive data: %d", id)
		stream.sendData = append(stream.sendData, buffer.NewIoBufferBytes(data).Clone())
		if endStream {
			conn.logger.Debugf("http2 server data: %d", id)
			stream.receiver.OnReceiveData(stream.ctx, stream.BuildData(), endStream)
		}
		return
	}

	// trailer
	if hasTrailer {
		if len(stream.sendData) > 0 {
			conn.logger.Debugf("http2 server data: %d", id)
			stream.receiver.OnReceiveData(stream.ctx, stream.BuildData(), false)
		}
		trailer := decodeHeader(h2s.Request.Trailer)
		conn.logger.Debugf("http2 server trailer: %d, %v", id, trailer)
		stream.receiver.OnReceiveTrailers(ctx, protocol.CommonHeader(trailer))
		return
	}

	// nil data
	if endStream {
		conn.logger.Debugf("http2 server data: %d", id)
		stream.receiver.OnReceiveData(stream.ctx, stream.BuildData(), endStream)
	}
}

func (conn *serverStreamConnection) handleError(ctx context.Context, f http2.Frame, err error) {
	conn.sc.HandleError(ctx, f, err)
	if err != nil {
		switch err := err.(type) {
		case http2.StreamError:
			conn.logger.Errorf("Http2 server handleError stream error: %v", err)
			conn.slock.Lock()
			delete(conn.streams, err.StreamID)
			conn.slock.Unlock()
		case http2.ConnectionError:
			conn.logger.Errorf("Http2 server handleError conn err: %v", err)
			conn.conn.Close(types.NoFlush, types.OnReadErrClose)
		default:
			conn.logger.Errorf("Http2 server handleError err: %v", err)
			conn.conn.Close(types.NoFlush, types.RemoteClose)
		}
	}
}

func (conn *serverStreamConnection) onNewStreamDetect(ctx context.Context, h2s *http2.MStream, endStream bool) (*serverStream, error) {
	stream := &serverStream{}
	stream.id = h2s.ID()
	stream.ctx = context.WithValue(ctx, types.ContextKeyStreamID, stream.id)
	stream.sc = conn
	stream.h2s = h2s
	stream.conn = conn.conn

	if !endStream {
		conn.slock.Lock()
		conn.streams[stream.id] = stream
		conn.slock.Unlock()
	}

	stream.receiver = conn.serverCallbacks.NewStreamDetect(stream.ctx, stream)
	return stream, nil
}

func (conn *serverStreamConnection) onStreamRecv(ctx context.Context, id uint32, endStream bool) *serverStream {
	conn.slock.Lock()
	defer conn.slock.Unlock()
	if stream, ok := conn.streams[id]; ok {
		if endStream {
			delete(conn.streams, id)
		}

		conn.logger.Debugf("http2 server OnStreamRecv, id = %d", stream.id)
		return stream
	}
	return nil
}

type serverStream struct {
	stream
	h2s *http2.MStream
	sc  *serverStreamConnection
}

// types.StreamSender
func (s *serverStream) AppendHeaders(ctx context.Context, headersIn types.HeaderMap, endStream bool) error {
	headers := headersIn.(protocol.CommonHeader)
	s.sc.logger.Debugf("http2 server ApppendHeaders id = %d, headers = %v", s.id, headers)

	rsp := new(http.Response)
	s.h2s.Response = rsp

	if value, ok := headers[types.HeaderStatus]; ok {
		rsp.StatusCode, _ = strconv.Atoi(value)
		delete(headers, types.HeaderStatus)
	} else {
		rsp.StatusCode = 200
	}

	rsp.Header = encodeHeader(headers)

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *serverStream) AppendData(context context.Context, data types.IoBuffer, endStream bool) error {
	s.h2s.SendData = data
	s.sc.logger.Debugf("http2 server ApppendData id = %d", s.id)

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *serverStream) AppendTrailers(context context.Context, trailers types.HeaderMap) error {
	headers := trailers.(protocol.CommonHeader)
	s.sc.logger.Debugf("http2 server ApppendTrailers id = %d, trailers = %v", s.id, headers)
	s.h2s.Response.Trailer = encodeHeader(headers)
	s.endStream()

	return nil
}

func (s *serverStream) endStream() {
	_, err := s.sc.codecEngine.Encode(s.ctx, s.h2s)
	if err != nil {
		s.sc.logger.Errorf("http2 server SendResponse  error :%v", err)
		s.stream.ResetStream(types.StreamLocalReset)
		return
	}
	s.sc.logger.Debugf("http2 server SendResponse id = %d", s.id)

}

func (s *serverStream) ResetStream(reason types.StreamResetReason) {
	// on stream reset
	s.sc.logger.Errorf("http2 server reset stream id = %d, error = %v", s.id, reason)
	s.h2s.Reset()
	s.stream.ResetStream(reason)
}

func (s *serverStream) GetStream() types.Stream {
	return s
}

type clientStreamConnection struct {
	streamConnection
	slock   sync.Mutex
	streams map[uint32]*clientStream
	cc      *http2.MClientConn

	clientCallbacks types.StreamConnectionEventListener
}

func newClientStreamConnection(ctx context.Context, connection types.Connection, clientCallbacks types.StreamConnectionEventListener) types.ClientStreamConnection {

	h2cc := http2.NewClientConn(connection)

	sc := &clientStreamConnection{
		streamConnection: streamConnection{
			ctx:         ctx,
			conn:        connection,
			codecEngine: mhttp2.EngineClient(h2cc),

			cm: contextManager{base: ctx},

			logger: log.ByContext(ctx),
		},
		cc:              h2cc,
		clientCallbacks: clientCallbacks,
	}

	// init first context
	sc.cm.next()

	sc.streams = make(map[uint32]*clientStream, 32)

	return sc
}

// types.StreamConnection
func (conn *clientStreamConnection) Dispatch(buf types.IoBuffer) {
	for {
		// 1. pre alloc stream-level ctx with bufferCtx
		ctx := conn.cm.curr

		// 2. decode process
		frame, err := conn.codecEngine.Decode(ctx, buf)
		// No enough data
		if err == http2.ErrAGAIN {
			break
		}

		// Do handle staff. Error would also be passed to this function.
		conn.handleFrame(ctx, frame, err)
		if err != nil {
			break
		}

		conn.cm.next()
	}

}

func (conn *clientStreamConnection) NewStream(ctx context.Context, receiver types.StreamReceiver) types.StreamSender {
	stream := &clientStream{}

	stream.ctx = ctx
	stream.sc = conn
	stream.receiver = receiver
	stream.conn = conn.conn
	stream.logger = log.ByContext(ctx)

	return stream
}

func (conn *clientStreamConnection) handleFrame(ctx context.Context, i interface{}, err error) {
	f, _ := i.(http2.Frame)
	if err != nil {
		conn.handleError(ctx, f, err)
		return
	}
	var endStream bool
	var data []byte
	var trailer http.Header
	var rsp *http.Response

	rsp, data, trailer, endStream, err = conn.cc.HandleFrame(ctx, f)

	if err != nil {
		conn.handleError(ctx, f, err)
		return
	}

	if rsp == nil && trailer == nil && data == nil && !endStream {
		return
	}

	id := f.Header().StreamID

	conn.slock.Lock()
	stream := conn.streams[id]
	if endStream && stream != nil {
		delete(conn.streams, id)
	}
	conn.slock.Unlock()

	if stream == nil {
		conn.logger.Errorf("http2 client invaild steamID :%v", f)
		return
	}

	if rsp != nil {
		header := decodeHeader(rsp.Header)

		header[types.HeaderStatus] = strconv.Itoa(rsp.StatusCode)

		buffer.TransmitBufferPoolContext(stream.ctx, ctx)

		stream.logger.Debugf("http2 client header: id = %d, headers = %v", id, header)
		stream.receiver.OnReceiveHeaders(ctx, protocol.CommonHeader(header), endStream)
		return
	}

	// data
	if data != nil {
		stream.logger.Debugf("http2 client receive data: id = %d", id)
		stream.sendData = append(stream.sendData, buffer.NewIoBufferBytes(data).Clone())
		if endStream {
			stream.logger.Debugf("http2 client data: id = %d", id)
			stream.receiver.OnReceiveData(stream.ctx, stream.BuildData(), endStream)
		}
		return
	}

	// trailer
	if trailer != nil {
		if len(stream.sendData) > 0 {
			stream.logger.Debugf("http2 client data: id = %d", id)
			stream.receiver.OnReceiveData(stream.ctx, stream.BuildData(), false)
		}
		trailers := decodeHeader(trailer)
		stream.logger.Debugf("http2 client trailer: id = %d, trailers = %v", id, trailers)
		stream.receiver.OnReceiveTrailers(ctx, protocol.CommonHeader(trailers))
		return
	}

	// nil data
	if endStream {
		stream.logger.Debugf("http2 client data: id = %d", id)
		stream.receiver.OnReceiveData(stream.ctx, stream.BuildData(), endStream)
	}
}

func (conn *clientStreamConnection) handleError(ctx context.Context, f http2.Frame, err error) {
	conn.cc.HandleError(ctx, f, err)
	if err != nil {
		switch err := err.(type) {
		case http2.StreamError:
			conn.logger.Errorf("Http2 client handleError stream err: %v", err)
			conn.slock.Lock()
			s := conn.streams[err.StreamID]
			if s != nil {
				delete(conn.streams, err.StreamID)
			}
			conn.slock.Unlock()
			if s != nil {
				s.ResetStream(types.StreamRemoteReset)
			}
		case http2.ConnectionError:
			conn.logger.Errorf("Http2 client handleError conn err: %v", err)
			conn.conn.Close(types.FlushWrite, types.OnReadErrClose)
		default:
			conn.logger.Errorf("Http2 client handleError err: %v", err)
			conn.conn.Close(types.NoFlush, types.RemoteClose)
		}
	}
}

type clientStream struct {
	stream
	logger log.Logger
	h2s    *http2.MClientStream
	sc     *clientStreamConnection
}

func (s *clientStream) AppendHeaders(ctx context.Context, headersIn types.HeaderMap, endStream bool) error {
	headersMap, _ := headersIn.(protocol.CommonHeader)

	req := new(http.Request)

	scheme := "http"
	if _, ok := s.conn.RawConn().(*mtls.TLSConn); ok {
		scheme = "https"
	}

	if method, ok := headersMap[types.HeaderMethod]; ok {
		delete(headersMap, types.HeaderMethod)
		req.Method = method
	} else {
		if endStream {
			req.Method = http.MethodGet
		} else {
			req.Method = http.MethodPost
		}
	}

	if host, ok := headersMap[types.HeaderHost]; ok {
		delete(headersMap, types.HeaderHost)
		req.Host = host
	} else if host, ok := headersMap["Host"]; ok {
		req.Host = host
	} else {
		req.Host = s.conn.RemoteAddr().String()
	}

	var URI string
	if path, ok := headersMap[types.HeaderPath]; ok {
		delete(headersMap, types.HeaderPath)
		URI = fmt.Sprintf(scheme+"://%s%s", req.Host, path)
		req.URL, _ = url.Parse(URI)
	}

	if _, ok := headersMap[types.HeaderStreamID]; ok {
		delete(headersMap, types.HeaderStreamID)
	}

	req.Header = encodeHeader(headersMap)

	s.logger.Debugf("http2 client AppendHeaders: id = %d, headers = %v", s.id, req.Header)

	s.h2s = http2.NewMClientStream(s.sc.cc, req)

	if endStream {
		s.endStream()
	}
	return nil
}

func (s *clientStream) AppendData(context context.Context, data types.IoBuffer, endStream bool) error {
	s.h2s.SendData = data
	s.logger.Debugf("http2 client AppendData: id = %d", s.id)
	if endStream {
		s.endStream()
	}

	return nil
}

func (s *clientStream) AppendTrailers(context context.Context, trailers types.HeaderMap) error {
	trailer := trailers.(protocol.CommonHeader)
	s.logger.Debugf("http2 client AppendTrailers: id = %d, trailers = %v", s.id, trailer)
	s.h2s.Request.Trailer = encodeHeader(trailer)
	s.endStream()

	return nil
}

func (s *clientStream) endStream() {
	s.sc.slock.Lock()
	defer s.sc.slock.Unlock()

	_, err := s.sc.codecEngine.Encode(s.ctx, s.h2s)
	if err != nil {
		s.sc.logger.Errorf("http2 client endStream error = :%v", err)
		s.ResetStream(types.StreamLocalReset)
		return
	}
	s.id = s.h2s.GetID()
	s.sc.streams[s.id] = s

	s.logger.Debugf("http2 client SendRequest id = %d", s.id)
}

func (s *clientStream) GetStream() types.Stream {
	return s
}

func encodeHeader(in map[string]string) (header http.Header) {
	header = http.Header((make(map[string][]string, len(in))))
	for k, v := range in {
		header.Add(k, v)
	}
	return header
}

func decodeHeader(in map[string][]string) (out map[string]string) {
	out = make(map[string]string, len(in))

	for k, v := range in {
		out[k] = strings.Join(v, ",")
	}
	return
}
