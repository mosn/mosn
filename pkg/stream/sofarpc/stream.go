package sofarpc

import (
	"errors"
	"github.com/orcaman/concurrent-map"
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

func (f *streamConnFactory) CreateClientStream(connection types.ClientConnection,
	clientCallbacks types.StreamConnectionEventListener, connCallbacks types.ConnectionEventListener) types.ClientStreamConnection {
	return newStreamConnection(connection, clientCallbacks, nil)
}

func (f *streamConnFactory) CreateServerStream(connection types.Connection,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	return newStreamConnection(connection, nil, serverCallbacks)
}

func (f *streamConnFactory) CreateBiDirectStream(connection types.ClientConnection, clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {
	return newStreamConnection(connection, clientCallbacks, serverCallbacks)
}

// types.DecodeFilter
// types.StreamConnection
// types.ClientStreamConnection
// types.ServerStreamConnection
type streamConnection struct {
	protocol        types.Protocol
	connection      types.Connection
	protocols       types.Protocols
	activeStream    cmap.ConcurrentMap // string: *stream
	clientCallbacks types.StreamConnectionEventListener
	serverCallbacks types.ServerStreamConnectionEventListener
}

func newStreamConnection(connection types.Connection, clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {

	return &streamConnection{
		connection:      connection,
		protocols:       sofarpc.DefaultProtocols(),
		activeStream:    cmap.New(),
		clientCallbacks: clientCallbacks,
		serverCallbacks: serverCallbacks,
	}
}

// types.StreamConnection
func (conn *streamConnection) Dispatch(buffer types.IoBuffer) {
	conn.protocols.Decode(buffer, conn)
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
	stream := &stream{
		streamId:   streamId,
		requestId:  streamId,
		direction:  OutStream,
		connection: conn,
		decoder:    responseDecoder,
	}
	conn.activeStream.Set(streamId, stream)

	return stream
}

func (conn *streamConnection) OnDecodeHeader(streamId string, headers map[string]string) types.FilterStatus {
	if sofarpc.IsSofaRequest(headers) || sofarpc.HasCodecException(headers) {
		conn.onNewStreamDetected(streamId, headers)
	}

	headers = decodeSterilize(streamId, headers)

	if v, ok := conn.activeStream.Get(streamId); ok {
		stream := v.(*stream)
		stream.decoder.OnDecodeHeaders(headers, false)
	}

	return types.Continue
}

func (conn *streamConnection) OnDecodeData(streamId string, data types.IoBuffer) types.FilterStatus {
	if v, ok := conn.activeStream.Get(streamId); ok {
		stream := v.(*stream)
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
	if v, ok := headers[sofarpc.SofaPropertyHeader("requestid")]; ok {
		requestId = v
	} else {
		// on decode exception stream
		requestId = streamId
	}

	headers[sofarpc.SofaPropertyHeader("requestid")] = streamId

	stream := &stream{
		streamId:   streamId,
		requestId:  requestId,
		direction:  InStream,
		connection: conn,
	}

	stream.decoder = conn.serverCallbacks.NewStream(streamId, stream)
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
		if v, ok := s.connection.activeStream.Get(s.streamId); ok {
			stream := v.(*stream)
			stream.connection.connection.Write(s.encodedHeaders)

			if s.encodedData != nil {
				stream.connection.connection.Write(s.encodedData)
			} else {
				log.DefaultLogger.Debugf("Response Body is void...")
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
