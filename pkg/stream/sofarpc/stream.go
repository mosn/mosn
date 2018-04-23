package sofarpc

import (
	"context"
	"errors"
	l "log"
	"sync"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	str "gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type StreamDirection int

const (
	InStream  StreamDirection = 1
	OutStream StreamDirection = 0
)

func init() {
	str.Register(protocol.SofaRpc, &streamConnFactory{})
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
	protocol        types.Protocol
	connection      types.Connection
	protocols       types.Protocols
	activeStream    streamMap
	clientCallbacks types.StreamConnectionEventListener
	serverCallbacks types.ServerStreamConnectionEventListener
}

func newStreamConnection(context context.Context, connection types.Connection, clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {

	return &streamConnection{
		connection:      connection,
		protocols:       sofarpc.DefaultProtocols(),
		activeStream:    newStreamMap(context),
		clientCallbacks: clientCallbacks,
		serverCallbacks: serverCallbacks,
	}
}

// types.StreamConnection
func (conn *streamConnection) Dispatch(buffer types.IoBuffer, context context.Context) {
	conn.protocols.Decode(context, buffer, conn)
}

func (conn *streamConnection) Protocol() types.Protocol {
	return conn.protocol
}

func (conn *streamConnection) OnUnderlyingConnectionAboveWriteBufferHighWatermark() {
	// todo
}

func (conn *streamConnection) OnUnderlyingConnectionBelowWriteBufferLowWatermark() {
	// todo
}

func (conn *streamConnection) NewStream(streamId string, responseDecoder types.StreamDecoder) types.StreamEncoder {
	stream := stream{
		streamId:   streamId,
		requestId:  streamId,
		direction:  OutStream,
		connection: conn,
		decoder:    responseDecoder,
	}
	conn.activeStream.Set(streamId, stream)

	return &stream
}

func (conn *streamConnection) OnDecodeHeader(streamId string, headers map[string]string) types.FilterStatus {
	if sofarpc.IsSofaRequest(headers) || sofarpc.HasCodecException(headers) {
		conn.onNewStreamDetected(streamId, headers)
	}

	decodeSterilize(streamId, headers)

	if stream, ok := conn.activeStream.Get(streamId); ok {
		stream.decoder.OnDecodeHeaders(headers, false)
	}

	return types.Continue
}

func (conn *streamConnection) OnDecodeData(streamId string, data types.IoBuffer) types.FilterStatus {
	if stream, ok := conn.activeStream.Get(streamId); ok {
		stream.decoder.OnDecodeData(data, true)

		if stream.direction == OutStream {
			stream.connection.activeStream.Remove(stream.streamId)
		}
	}

	return types.StopIteration
}

func (conn *streamConnection) OnDecodeTrailer(streamId string, trailers map[string]string) types.FilterStatus {
	// unsupported
	return types.StopIteration
}

func (conn *streamConnection) onNewStreamDetected(streamId string, headers map[string]string) {
	if ok := conn.activeStream.Has(streamId); ok {
		return
	}

	var requestId string
	if v, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)]; ok {
		requestId = v
	} else {
		// on decode exception stream
		requestId = streamId
	}

	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = streamId

	stream := stream{
		streamId:   streamId,
		requestId:  requestId,
		direction:  InStream,
		connection: conn,
	}

	stream.decoder = conn.serverCallbacks.NewStream(streamId, &stream)
	conn.activeStream.Set(streamId, stream)
}

// types.Stream
// types.StreamEncoder
type stream struct {
	streamId         string
	requestId        string
	direction        StreamDirection // 0: out, 1: in
	readDisableCount int
	connection       *streamConnection
	decoder          types.StreamDecoder
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

// types.StreamEncoder
func (s *stream) EncodeHeaders(headers interface{}, endStream bool) error {
	_, s.encodedHeaders = s.connection.protocols.EncodeHeaders(s.encodeSterilize(headers))

	// Exception occurs when encoding headers
	if s.encodedHeaders == nil {
		return errors.New(s.streamId)
	}

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *stream) EncodeData(data types.IoBuffer, endStream bool) error {
	s.encodedData = data

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *stream) EncodeTrailers(trailers map[string]string) error {
	s.endStream()

	return nil
}

func (s *stream) endStream() {
	if s.encodedHeaders != nil {
		if stream, ok := s.connection.activeStream.Get(s.streamId); ok {

			if s.encodedData != nil {
				stream.connection.connection.Write(s.encodedHeaders, s.encodedData)
			} else {
				log.DefaultLogger.Debugf("Response Body is void...")

				stream.connection.connection.Write(s.encodedHeaders)
			}
		} else {
			log.DefaultLogger.Errorf("No stream %s to end", s.streamId)
		}
	} else {
		log.DefaultLogger.Debugf("Response Headers is void...")
	}

	if s.direction == InStream {
		s.connection.activeStream.Remove(s.streamId)
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
	smap := str.GetMap(context, 5096)

	return streamMap{
		smap: smap,
	}
}

func (m *streamMap) Has(streamId string) bool {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if _, ok := m.smap[streamId]; ok {
		return true
	} else {
		return false
	}
}

func (m *streamMap) Get(streamId string) (stream, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if s, ok := m.smap[streamId]; ok {
		return s.(stream), ok
	} else {
		return stream{}, false
	}
}

func (m *streamMap) Remove(streamId string) {
	m.mux.Lock()
	defer m.mux.Unlock()

	delete(m.smap, streamId)
}

func (m *streamMap) Set(streamId string, s stream) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.smap[streamId] = s
}
