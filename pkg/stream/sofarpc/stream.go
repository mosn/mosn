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

package sofarpc

import (
	"context"
	"sync"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// StreamDirection represent the stream's direction
type StreamDirection int

// ServerStream = 1
// ClientStream = 0
const (
	ServerStream StreamDirection = 1
	ClientStream StreamDirection = 0
)

func init() {
	str.Register(protocol.SofaRPC, &streamConnFactory{})
}

type streamConnFactory struct{}

func (f *streamConnFactory) CreateClientStream(context context.Context, connection types.ClientConnection,
	clientCallbacks types.StreamConnectionEventListener, connCallbacks types.ConnectionEventListener) types.ClientStreamConnection {
	return newStreamConnection(context, connection, clientCallbacks, nil)
}

func (f *streamConnFactory) CreateServerStream(context context.Context, connection types.Connection,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	return newStreamConnection(context, connection, nil, serverCallbacks)
}

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
	protocols       types.Protocols
	activeStreams   streamMap
	clientCallbacks types.StreamConnectionEventListener
	serverCallbacks types.ServerStreamConnectionEventListener

	logger log.Logger
}

func newStreamConnection(context context.Context, connection types.Connection, clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {

	return &streamConnection{
		context:         context,
		connection:      connection,
		protocols:       sofarpc.DefaultProtocols(),
		activeStreams:   newStreamMap(context),
		clientCallbacks: clientCallbacks,
		serverCallbacks: serverCallbacks,
		logger:          log.ByContext(context),
	}
}

// types.StreamConnection
func (conn *streamConnection) Dispatch(buf types.IoBuffer) {
	conn.protocols.Decode(conn.context, buf, conn)
}

func (conn *streamConnection) Protocol() types.Protocol {
	return conn.protocol
}

func (conn *streamConnection) GoAway() {
	// todo
}

func (conn *streamConnection) NewStream(text context.Context, streamID string, responseDecoder types.StreamReceiver) types.StreamSender {
	sofabuffers := sofaBuffersByContent(text)
	stream := &sofabuffers.client
	stream.context = context.WithValue(text, types.ContextKeyStreamID, streamID)
	stream.streamID = streamID
	stream.requestID = streamID
	stream.direction = ClientStream
	stream.connection = conn
	stream.decoder = responseDecoder
	conn.activeStreams.Set(streamID, stream)

	return stream
}

func (conn *streamConnection) OnDecodeHeader(streamID string, headers map[string]string) types.FilterStatus {
	if sofarpc.IsSofaRequest(headers) {
		conn.onNewStreamDetected(streamID, headers)
	}
	endStream := decodeSterilize(streamID, headers)

	if stream, ok := conn.activeStreams.Get(streamID); ok {
		if conn.clientCallbacks != nil {
			buffer.CopyBufferPoolContext(stream.context, conn.context)
		}
		stream.decoder.OnReceiveHeaders(headers, endStream)
		if endStream {
			return types.StopIteration
		}
	}

	return types.Continue
}

func (conn *streamConnection) OnDecodeData(streamID string, data types.IoBuffer) types.FilterStatus {
	if stream, ok := conn.activeStreams.Get(streamID); ok {
		if stream.direction == ClientStream {
			// for client stream, remove stream on response read
			stream.connection.activeStreams.Remove(stream.streamID)
		}
		stream.decoder.OnReceiveData(data, true)
	}

	return types.StopIteration
}

func (conn *streamConnection) OnDecodeTrailer(streamID string, trailers map[string]string) types.FilterStatus {
	// unsupported
	return types.StopIteration
}

// todo, deal with more exception
func (conn *streamConnection) OnDecodeError(err error, header map[string]string) {
	if err == nil {
		return
	}

	switch err.Error() {
	case types.UnSupportedProCode, sofarpc.UnKnownCmdcode, sofarpc.UnKnownReqtype:
		// for header decode error, close the connection directly
		conn.connection.Close(types.NoFlush, types.LocalClose)
	case types.CodecException:
		if v, ok := header[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)]; ok {
			conn.onNewStreamDetected(v, header)

			if stream, ok := conn.activeStreams.Get(v); ok {
				if stream.direction == ClientStream {
					// for client stream, remove stream on response read
					stream.connection.activeStreams.Remove(stream.streamID)
				}
				stream.decoder.OnDecodeError(err, header)
			}
		} else {
			// if no request id found, no reason to send response, so close connection
			conn.connection.Close(types.NoFlush, types.LocalClose)
		}
	}
}

func (conn *streamConnection) onNewStreamDetected(streamID string, headers map[string]string) {
	if ok := conn.activeStreams.Has(streamID); ok {
		log.DefaultLogger.Infof("OnReceiveHeaders, stream already exist, maybe response, StreamID = %s", streamID)
		return
	}

	var requestID string
	if v, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)]; ok {
		requestID = v
	} else {
		// on decode exception stream
		requestID = streamID
	}

	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = streamID

	newcontext := buffer.NewBufferPoolContext(conn.context, true)

	sofabuffers := sofaBuffersByContent(newcontext)
    stream := &sofabuffers.server
	stream.context = context.WithValue(newcontext, types.ContextKeyStreamID, streamID)
	stream.streamID = streamID
	stream.requestID = requestID
	stream.direction = ServerStream
	stream.connection = conn

	log.DefaultLogger.Infof("OnReceiveHeaders, New stream detected, Request id = %s, StreamID = %s", requestID, streamID)

	stream.decoder = conn.serverCallbacks.NewStream(stream.context, streamID, stream)
	conn.activeStreams.Set(streamID, stream)
}

// types.Stream
// types.StreamSender
type stream struct {
	context context.Context

	buffers          *sofaBuffers

	streamID         string
	requestID        string
	direction        StreamDirection // 0: out, 1: in
	readDisableCount int
	connection       *streamConnection
	decoder          types.StreamReceiver
	streamCbs        []types.StreamEventListener
	encodedHeaders   types.IoBuffer
	encodedData      types.IoBuffer
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
	s.connection.connection.SetReadDisable(disable)
}

func (s *stream) BufferLimit() uint32 {
	return s.connection.connection.BufferLimit()
}

// types.StreamSender
func (s *stream)  AppendHeaders(headers interface{}, endStream bool) error {
	var err error

	if s.encodedHeaders, err = s.connection.protocols.EncodeHeaders(s.context, s.encodeSterilize(headers)); err != nil {
		return err
	}

	log.DefaultLogger.Infof("AppendHeaders,request id = %s, direction = %d", s.streamID, s.direction)

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *stream) AppendData(data types.IoBuffer, endStream bool) error {
	s.encodedData = data

	log.DefaultLogger.Infof("AppendData,request id = %s, direction = %d", s.streamID, s.direction)

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *stream) AppendTrailers(trailers map[string]string) error {
	s.endStream()

	return nil
}

// Flush stream data
// For server stream, write out response
// For client stream, write out request
func (s *stream) endStream() {
	if s.encodedHeaders != nil {
	//	log.DefaultLogger.Infof("Write to remote, stream id = %s, direction = %d", s.streamID, s.direction)

		if stream, ok := s.connection.activeStreams.Get(s.streamID); ok {

			if s.encodedData != nil {
				stream.connection.connection.Write(s.encodedHeaders, s.encodedData)
			} else {
				//	s.connection.logger.Debugf("stream %s response body is void...", s.streamID)
				stream.connection.connection.Write(s.encodedHeaders)
			}
		} else {
			s.connection.logger.Errorf("No stream %s to end", s.streamID)
		}
	} else {
		s.connection.logger.Debugf("Response Headers is void...")
	}

	if s.direction == ServerStream {
		// for a server stream, remove stream on response wrote
		s.connection.activeStreams.Remove(s.streamID)
		//	log.StartLogger.Warnf("Remove Request ID = %+v",s.streamID)
	}
}

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

func (m *streamMap) Has(streamID string) bool {
	m.mux.RLock()
	if _, ok := m.smap[streamID]; ok {
		m.mux.RUnlock()
		return true
	}
	m.mux.RUnlock()
	return false
}

func (m *streamMap) Get(streamID string) (*stream, bool) {
	m.mux.RLock()

	if s, ok := m.smap[streamID]; ok {
		m.mux.RUnlock()
		return s.(*stream), ok
	}
	m.mux.RUnlock()

	return &stream{}, false
}

func (m *streamMap) Remove(streamID string) {
	m.mux.Lock()
	delete(m.smap, streamID)
	m.mux.Unlock()

}

func (m *streamMap) Set(streamID string, s *stream) {
	m.mux.Lock()
	m.smap[streamID] = s
	m.mux.Unlock()
}
