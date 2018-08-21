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
	"strings"
	"sync"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	networkbuffer "github.com/alipay/sofa-mosn/pkg/network/buffer"
	"sync/atomic"
	"strconv"
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

// types.DecodeFilter
// types.StreamConnection
// types.ClientStreamConnection
// types.ServerStreamConnection
type streamConnection struct {
	context         context.Context
	protocol        types.Protocol
	connection      types.Connection
	activeStream    streamMap
	clientCallbacks types.StreamConnectionEventListener
	serverCallbacks types.ServerStreamConnectionEventListener
	codec 			types.Multiplexing
	streamIdMap 	sync.Map
	reqIdMap 		sync.Map
	logger log.Logger
}

func newStreamConnection(context context.Context, connection types.Connection, clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {
	subProtocolName := context.Value("XSubProtocol").(types.SubProtocol)
	codec := subProtocolFactories[subProtocolName].CreateSubProtocolCodec(context)
	return &streamConnection{
		context:         context,
		connection:      connection,
		activeStream:    newStreamMap(context),
		clientCallbacks: clientCallbacks,
		serverCallbacks: serverCallbacks,
		logger:          log.ByContext(context),
		codec: 			 codec,
	}
}

// Dispatch would invoked in this two situation:
// serverStreamConnection receive request
// clientStreamConnection receive response
// types.StreamConnection
func (conn *streamConnection) Dispatch(buffer types.IoBuffer) {
	log.DefaultLogger.Tracef("stream connection dispatch data = %v", buffer.String())
	headers := make(map[string]string)
	// support dynamic route
	headers[strings.ToLower(protocol.MosnHeaderHostKey)] = conn.connection.RemoteAddr().String()
	headers[strings.ToLower(protocol.MosnHeaderPathKey)] = "/"
	log.DefaultLogger.Tracef("before Dispatch on decode header")

	// get sub protocol codec
	_,requestList := conn.codec.SplitRequest(buffer.Bytes())
	for _,request := range requestList{
		requestLen := len(request)
		// ProtocolConvertor
		// convertor first
		convertorCodec,ok := conn.codec.(types.ProtocolConvertor)
		if ok {
			newHeaders , newData := convertorCodec.Convert(request)
			request = newData
			headers = newHeaders
		}

		// get stream id
		streamId := ""
		if conn.serverCallbacks != nil{
			// replace request id
			reqId := conn.codec.GetStreamId(request)
			streamId = conn.changeStreamId(&request)
			conn.reqIdMap.Store(streamId,reqId)
			log.DefaultLogger.Tracef("Xprotocol get streamId %v, old reqId = %v",streamId,reqId)

			// request route
			requestRouteCodec,ok := conn.codec.(types.RequestRouting)
			if ok {
				routeHeaders := requestRouteCodec.GetMetas(request)
				for k,v := range routeHeaders{
					headers[k] = v
				}
				log.DefaultLogger.Tracef("xprotocol handle request route ,headers = %v" , headers)
			}
		}else if conn.clientCallbacks != nil{
			tmpStreamId := conn.codec.GetStreamId(request)
			value,ok := conn.streamIdMap.Load(tmpStreamId)
			if ok{
				streamId = value.(string)
			}else{
				log.DefaultLogger.Tracef("fail to get old streamid , maybe streamid is changed by upstream server?")
			}
		}
		// tracing
		tracingCodec ,ok:= conn.codec.(types.Tracing)
		if ok {
			serviceName := tracingCodec.GetServiceName(request)
			methodName := tracingCodec.GetMethodName(request)
			headers[types.HeaderRpcService] = serviceName
			headers[types.HeaderRpcMethod] = methodName
			log.DefaultLogger.Tracef("xprotocol handle tracing ,serviceName = %v , methodName = %v",serviceName,methodName)
		}

		reqBuf := networkbuffer.NewIoBufferBytes(request)
		conn.OnReceiveHeaders(streamId, headers)
		log.DefaultLogger.Tracef("after Dispatch on decode header")
		conn.OnReceiveData(streamId, reqBuf)
		log.DefaultLogger.Tracef("after Dispatch on decode data")
		buffer.Drain(requestLen)
	}
}

func (conn *streamConnection) changeStreamId(request *[]byte) string{
	nStreamId := atomic.AddUint64(&streamIDXprotocolCount, 1)
	streamId  := strconv.FormatUint(nStreamId, 10)
	conn.codec.SetStreamId(request,streamId)
	streamId = conn.codec.GetStreamId(*request)
	return streamId
}
// Protocol return xprotocol
func (conn *streamConnection) Protocol() types.Protocol {
	return conn.protocol
}

func (conn *streamConnection) GoAway() {
	// todo
}

func (conn *streamConnection) OnUnderlyingConnectionAboveWriteBufferHighWatermark() {
	// todo
}

func (conn *streamConnection) OnUnderlyingConnectionBelowWriteBufferLowWatermark() {
	// todo
}

// NewStream
func (conn *streamConnection) NewStream(streamID string, responseDecoder types.StreamReceiver) types.StreamSender {
	log.DefaultLogger.Tracef("xprotocol stream new stream,streamId =%v ", streamID)
	stream := stream{
		context:    context.WithValue(conn.context, types.ContextKeyStreamID, streamID),
		streamID:   streamID,
		direction:  ClientStream,
		connection: conn,
		decoder:    responseDecoder,
	}
	conn.activeStream.Set(streamID, stream)

	return &stream
}

// OnReceiveHeaders process header
func (conn *streamConnection) OnReceiveHeaders(streamID string, headers map[string]string) types.FilterStatus {
	log.StartLogger.Tracef("xprotocol stream on decode header")
	if conn.serverCallbacks != nil {
		log.StartLogger.Tracef("xprotocol stream on new stream detected invoked")
		conn.onNewStreamDetected(streamID, headers)
	}
	if stream, ok := conn.activeStream.Get(streamID); ok {
		log.StartLogger.Tracef("before stream decoder invoke on decode header")
		stream.decoder.OnReceiveHeaders(headers, false)
	}
	log.StartLogger.Tracef("after stream decoder invoke on decode header")
	return types.Continue
}

// OnReceiveData process data
func (conn *streamConnection) OnReceiveData(streamID string, data types.IoBuffer) types.FilterStatus {
	if stream, ok := conn.activeStream.Get(streamID); ok {
		if stream.direction == ClientStream {
			// restore request id
			buf := data.Bytes()
			conn.codec.SetStreamId(&buf,stream.reqId)
			data = networkbuffer.NewIoBufferBytes(buf)
		}

		log.DefaultLogger.Tracef("xprotocol stream on decode data")
		stream.decoder.OnReceiveData(data, true)

		if stream.direction == ClientStream {
			// for client stream, remove stream on response read
			stream.connection.activeStream.Remove(stream.streamID)
		}
	}
	return types.StopIteration
}

func (conn *streamConnection) onNewStreamDetected(streamID string, headers map[string]string) {
	if ok := conn.activeStream.Has(streamID); ok {
		return
	}
	stream := stream{
		context:    context.WithValue(conn.context, types.ContextKeyStreamID, streamID),
		streamID:   streamID,
		direction:  ServerStream,
		connection: conn,
	}

	stream.decoder = conn.serverCallbacks.NewStream(streamID, &stream)
	conn.activeStream.Set(streamID, stream)
}

// types.Stream
// types.StreamEncoder
type stream struct {
	context context.Context

	streamID         string
	direction        StreamDirection // 0: out, 1: in
	readDisableCount int
	connection       *streamConnection
	decoder          types.StreamReceiver
	streamCbs        []types.StreamEventListener
	encodedHeaders   types.IoBuffer
	encodedData      types.IoBuffer
	reqId 			 string
}

// AddEventListener add stream event callback
// types.Stream
func (s *stream) AddEventListener(cb types.StreamEventListener) {
	s.streamCbs = append(s.streamCbs, cb)
}

// RemoveEventListener remove stream event callback
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

// ResetStream reset stream
func (s *stream) ResetStream(reason types.StreamResetReason) {
	for _, cb := range s.streamCbs {
		cb.OnResetStream(reason)
	}
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
func (s *stream) AppendHeaders(headers interface{}, endStream bool) error {
	log.StartLogger.Tracef("EncodeHeaders,request id = %s, direction = %d", s.streamID, s.direction)
	if endStream {
		s.endStream()
	}
	return nil
}

// AppendData process upstream request data
func (s *stream) AppendData(data types.IoBuffer, endStream bool) error {
	if s.direction == ClientStream {
		buf := data.Bytes()
		s.reqId = s.connection.codec.GetStreamId(buf)
		streamId := s.connection.changeStreamId(&buf)
		reqBuf := networkbuffer.NewIoBufferBytes(buf)
		// save streamid mapping dict
		s.connection.streamIdMap.Store(streamId, s.streamID)
		s.encodedData = reqBuf
	}else if s.direction == ServerStream {
		streamId := s.connection.codec.GetStreamId(data.Bytes())
		value,ok := s.connection.reqIdMap.Load(streamId)
		if ok{
			// restore request id
			reqId := value.(string)
			buf := data.Bytes()
			s.connection.codec.SetStreamId(&buf,reqId)
			reqBuf := networkbuffer.NewIoBufferBytes(buf)
			s.encodedData = reqBuf
		}
	}
	log.DefaultLogger.Tracef("EncodeData,request id = %s, direction = %d,data = %v", s.streamID, s.direction, data.String())
	if endStream {
		s.endStream()
	}
	return nil
}

// AppendTrailers process upstream request trailers
func (s *stream) AppendTrailers(trailers map[string]string) error {
	log.DefaultLogger.Tracef("EncodeTrailers,request id = %s, direction = %d", s.streamID, s.direction)
	s.endStream()
	return nil
}

// Flush stream data
// For server stream, write out response
// For client stream, write out request

//TODO: x-subprotocol stream has encodeHeaders?
func (s *stream) endStream() {
	log.StartLogger.Tracef("xprotocol stream end stream invoked , request id = %s, direction = %d", s.streamID, s.direction)
	if stream, ok := s.connection.activeStream.Get(s.streamID); ok {
		log.StartLogger.Tracef("xprotocol stream end stream write encodedata")
		stream.connection.connection.Write(s.encodedData)
	} else {
		s.connection.logger.Errorf("No stream %s to end", s.streamID)
	}

	if s.direction == ServerStream {
		// for a server stream, remove stream on response wrote
		s.connection.activeStream.Remove(s.streamID)
		log.StartLogger.Warnf("Remove Request ID = %+v", s.streamID)
	}
}

// GetStream return stream
func (s *stream) GetStream() types.Stream {
	return s
}

type streamMap struct {
	smap map[string]interface{}
	mux  sync.RWMutex
}

func newStreamMap(context context.Context) streamMap {
	smap := make(map[string]interface{}, 32)

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
		return s.(stream), ok
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

