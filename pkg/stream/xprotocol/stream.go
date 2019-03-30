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

package xprotocol

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	networkbuffer "github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/xprotocol"
	_ "github.com/alipay/sofa-mosn/pkg/protocol/rpc/xprotocol/dubbo"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var streamIDXprotocolCount uint64

// StreamDirection 1: server stream 0: client stream
type StreamDirection int

const (
	//ServerStream xprotocol as downstream
	ServerStream StreamDirection = 1
	//ClientStream xprotocol as upstream
	ClientStream StreamDirection = 0
)

func init() {
	str.Register(protocol.Xprotocol, &streamConnFactory{})
}

type streamConnFactory struct{}

// CreateClientStream upstream create
func (f *streamConnFactory) CreateClientStream(context context.Context, connection types.ClientConnection,
	clientCallbacks types.StreamConnectionEventListener, connCallbacks types.ConnectionEventListener) types.ClientStreamConnection {
	return newStreamConnection(context, connection, clientCallbacks, nil)
}

// CreateServerStream downstream create
func (f *streamConnFactory) CreateServerStream(context context.Context, connection types.Connection,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	return newStreamConnection(context, connection, nil, serverCallbacks)
}

// CreateBiDirectStream no used
func (f *streamConnFactory) CreateBiDirectStream(context context.Context, connection types.ClientConnection,
	clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {
	return newStreamConnection(context, connection, clientCallbacks, serverCallbacks)
}

func (f *streamConnFactory) ProtocolMatch(prot string, magic []byte) error {
	return str.FAILED
}

// types.DecodeFilter
// types.StreamConnection
// types.ClientStreamConnection
// types.ServerStreamConnection
type streamConnection struct {
	context                             context.Context
	protocol                            types.Protocol
	connection                          types.Connection
	activeStream                        streamMap
	codec                               xprotocol.Multiplexing
	logger                              log.ErrorLogger
	streamConnectionEventListener       types.StreamConnectionEventListener
	serverStreamConnectionEventListener types.ServerStreamConnectionEventListener
}

func newStreamConnection(context context.Context, connection types.Connection, clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {
	subProtocolName := xprotocol.SubProtocol(context.Value(types.ContextSubProtocol).(string))
	log.DefaultLogger.Tracef("xprotocol subprotocol config name = %v", subProtocolName)
	codec := xprotocol.CreateSubProtocolCodec(context, subProtocolName)
	log.DefaultLogger.Tracef("xprotocol new stream connection, codec type = %v", subProtocolName)
	return &streamConnection{
		context:                             context,
		connection:                          connection,
		activeStream:                        newStreamMap(context),
		streamConnectionEventListener:       clientCallbacks,
		serverStreamConnectionEventListener: serverCallbacks,
		logger:   log.ByContext(context),
		codec:    codec,
		protocol: protocol.Xprotocol,
	}
}

// Dispatch would invoked in this two situation:
// serverStreamConnection receive request
// clientStreamConnection receive response
// types.StreamConnection
func (conn *streamConnection) Dispatch(buffer types.IoBuffer) {
	log.DefaultLogger.Tracef("stream connection dispatch data bytes = %v", buffer.Bytes())
	log.DefaultLogger.Tracef("stream connection dispatch data string = %v", buffer.String())

	// get sub protocol codec
	requestList := conn.codec.SplitFrame(buffer.Bytes())
	for _, request := range requestList {
		headers := make(map[string]string)
		// support dynamic route
		headers[strings.ToLower(protocol.MosnHeaderHostKey)] = conn.connection.RemoteAddr().String()
		headers[strings.ToLower(protocol.MosnHeaderPathKey)] = "/"
		log.DefaultLogger.Tracef("before Dispatch on decode header")

		requestLen := len(request)
		// ProtocolConvertor
		// convertor first
		convertorCodec, ok := conn.codec.(xprotocol.ProtocolConvertor)
		if ok {
			newHeaders, newData := convertorCodec.Convert(request)
			request = newData
			headers = newHeaders
		}

		// get stream id
		streamID := conn.codec.GetStreamID(request)
		if conn.serverStreamConnectionEventListener != nil {
			log.DefaultLogger.Tracef("Xprotocol get streamId %v", streamID)

			// request route
			requestRouteCodec, ok := conn.codec.(xprotocol.RequestRouting)
			if ok {
				routeHeaders := requestRouteCodec.GetMetas(request)
				for k, v := range routeHeaders {
					headers[k] = v
				}
				log.DefaultLogger.Tracef("xprotocol handle request route ,headers = %v", headers)
			}
		}
		// tracing
		tracingCodec, ok := conn.codec.(xprotocol.Tracing)
		if ok {
			serviceName := tracingCodec.GetServiceName(request)
			methodName := tracingCodec.GetMethodName(request)
			headers[types.HeaderRPCService] = serviceName
			headers[types.HeaderRPCMethod] = methodName
			log.DefaultLogger.Tracef("xprotocol handle tracing ,serviceName = %v , methodName = %v", serviceName, methodName)
		}

		reqBuf := networkbuffer.NewIoBufferBytes(request)
		log.DefaultLogger.Tracef("after Dispatch on decode header and data")
		conn.OnReveive(conn.context, streamID, protocol.CommonHeader(headers), reqBuf)
		buffer.Drain(requestLen)
	}
}

// Protocol return xprotocol
func (conn *streamConnection) Protocol() types.Protocol {
	return conn.protocol
}

func (conn *streamConnection) GoAway() {
	// unsupported
}

func (conn *streamConnection) ActiveStreamsNum() int {
	return conn.activeStream.Len()
}

func (conn *streamConnection) Reset(reason types.StreamResetReason) {
	conn.activeStream.mux.Lock()
	defer conn.activeStream.mux.Unlock()

	for _, s := range conn.activeStream.smap {
		s.ResetStream(reason)
	}
}

// NewStream
func (conn *streamConnection) NewStream(ctx context.Context, responseDecoder types.StreamReceiveListener) types.StreamSender {
	nStreamID := atomic.AddUint64(&streamIDXprotocolCount, 1)
	streamID := strconv.FormatUint(nStreamID, 10)

	stream := stream{
		context:        context.WithValue(ctx, types.ContextKeyStreamID, streamID),
		streamID:       streamID,
		direction:      ClientStream,
		connection:     conn,
		streamReceiver: responseDecoder,
	}
	conn.activeStream.Set(streamID, stream)

	return &stream
}

func (conn *streamConnection) OnReveive(context context.Context, streamID string, headers types.HeaderMap, data types.IoBuffer) types.FilterStatus {
	log.DefaultLogger.Tracef("xprotocol stream on decode header")
	if conn.serverStreamConnectionEventListener != nil {
		log.DefaultLogger.Tracef("xprotocol stream on new stream detected invoked")
		conn.onNewStreamDetected(streamID, headers)
	}
	if stream, ok := conn.activeStream.Get(streamID); ok {
		log.DefaultLogger.Tracef("xprotocol stream on decode header and data")
		stream.streamReceiver.OnReceive(context, headers, data, nil)

		if stream.direction == ClientStream {
			// for client stream, remove stream on response read
			stream.connection.activeStream.Remove(stream.streamID)
		}
	}
	return types.Stop
}

// OnReceiveHeaders process header
func (conn *streamConnection) OnReceiveHeaders(context context.Context, streamID string, headers types.HeaderMap) types.FilterStatus {
	log.DefaultLogger.Tracef("xprotocol stream on decode header")
	if conn.serverStreamConnectionEventListener != nil {
		log.DefaultLogger.Tracef("xprotocol stream on new stream detected invoked")
		conn.onNewStreamDetected(streamID, headers)
	}
	if stream, ok := conn.activeStream.Get(streamID); ok {
		log.DefaultLogger.Tracef("before stream streamReceiver invoke on decode header")
		stream.streamReceiver.OnReceiveHeaders(context, headers, false)
	}
	log.DefaultLogger.Tracef("after stream streamReceiver invoke on decode header")
	return types.Continue
}

// OnReceiveData process data
func (conn *streamConnection) OnReceiveData(context context.Context, streamID string, data types.IoBuffer) types.FilterStatus {
	if stream, ok := conn.activeStream.Get(streamID); ok {
		log.DefaultLogger.Tracef("xprotocol stream on decode data")
		stream.streamReceiver.OnReceiveData(context, data, true)

		if stream.direction == ClientStream {
			// for client stream, remove stream on response read
			stream.connection.activeStream.Remove(stream.streamID)
		}
	}
	return types.Stop
}

func (conn *streamConnection) onNewStreamDetected(streamID string, headers types.HeaderMap) {
	if ok := conn.activeStream.Has(streamID); ok {
		return
	}
	stream := stream{
		context:    context.WithValue(conn.context, types.ContextKeyStreamID, streamID),
		streamID:   streamID,
		direction:  ServerStream,
		connection: conn,
	}

	stream.streamReceiver = conn.serverStreamConnectionEventListener.NewStreamDetect(conn.context, &stream, nil)
	conn.activeStream.Set(streamID, stream)
}

// types.Stream
// types.StreamEncoder
type stream struct {
	str.BaseStream

	reqID            string
	streamID         string
	direction        StreamDirection // 0: out, 1: in
	readDisableCount int
	context          context.Context
	connection       *streamConnection
	streamReceiver   types.StreamReceiveListener
	encodedHeaders   types.IoBuffer
	encodedData      types.IoBuffer
}

// AddEventListener add stream event callback
// types.Stream
func (s *stream) ID() uint64 {
	id, _ := strconv.ParseUint(s.streamID, 10, 64)
	return id
}

// ReadDisable disable the read loop goroutine on connection
func (s *stream) ReadDisable(disable bool) {
	s.connection.connection.SetReadDisable(disable)
}

// BufferLimit buffer limit
func (s *stream) BufferLimit() uint32 {
	return s.connection.connection.BufferLimit()
}

// AppendHeaders process upstream request header
// types.StreamEncoder
func (s *stream) AppendHeaders(context context.Context, headers types.HeaderMap, endStream bool) error {
	log.DefaultLogger.Tracef("EncodeHeaders,request id = %s, direction = %d", s.streamID, s.direction)
	if endStream {
		s.endStream()
	}
	return nil
}

// AppendData process upstream request data
func (s *stream) AppendData(context context.Context, data types.IoBuffer, endStream bool) error {
	// replace request id
	newData := s.connection.codec.SetStreamID(data.Bytes(), s.streamID)
	s.encodedData = networkbuffer.NewIoBufferBytes(newData)

	if endStream {
		s.endStream()
	}
	return nil
}

// AppendTrailers process upstream request trailers
func (s *stream) AppendTrailers(context context.Context, trailers types.HeaderMap) error {
	log.DefaultLogger.Tracef("EncodeTrailers,request id = %s, direction = %d", s.streamID, s.direction)
	s.endStream()
	return nil
}

// Flush stream data
// For server stream, write out response
// For client stream, write out request

//TODO: x-subprotocol stream has encodeHeaders?
func (s *stream) endStream() {
	defer func() {
		if s.direction == ServerStream {
			s.DestroyStream()
		}
	}()

	log.DefaultLogger.Tracef("xprotocol stream end stream invoked , request id = %s, direction = %d", s.streamID, s.direction)
	if stream, ok := s.connection.activeStream.Get(s.streamID); ok {
		log.DefaultLogger.Tracef("xprotocol stream end stream write encodedata = %v", s.encodedData)
		stream.connection.connection.Write(s.encodedData)
	} else {
		s.connection.logger.Errorf("No stream %s to end", s.streamID)
	}

	if s.direction == ServerStream {
		// for a server stream, remove stream on response wrote
		s.connection.activeStream.Remove(s.streamID)
		log.DefaultLogger.Tracef("Remove Request ID = %+v", s.streamID)
	}
}

// GetStream return stream
func (s *stream) GetStream() types.Stream {
	return s
}

type streamMap struct {
	smap map[string]stream
	mux  sync.RWMutex
}

func newStreamMap(context context.Context) streamMap {
	smap := make(map[string]stream, 32)

	return streamMap{
		smap: smap,
	}
}

// Has check stream id
func (m *streamMap) Has(streamID string) bool {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if _, ok := m.smap[streamID]; ok {
		return true
	}

	return false
}

// Get return stream
func (m *streamMap) Get(streamID string) (stream, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if s, ok := m.smap[streamID]; ok {
		return s, ok
	}

	return stream{}, false
}

// Remove delete stream
func (m *streamMap) Remove(streamID string) {
	m.mux.Lock()
	defer m.mux.Unlock()

	delete(m.smap, streamID)
}

// Set add stream
func (m *streamMap) Set(streamID string, s stream) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.smap[streamID] = s
}

func (m *streamMap) Len() int {
	m.mux.Lock()
	defer m.mux.Unlock()

	return len(m.smap)
}
