package sofarpc

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/proxy"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
)

func init() {
	proxy.Register(protocol.SofaRpc, &streamConnFactory{})
}

type streamConnFactory struct{}

func (f *streamConnFactory) CreateClientStream(connection types.ClientConnection,
	streamConnCallbacks types.StreamConnectionCallbacks, connCallbacks types.ConnectionCallbacks) types.ClientStreamConnection {
	return newClientStreamConnection(connection, streamConnCallbacks)
}

func (f *streamConnFactory) CreateServerStream(connection types.ServerConnection,
	callbacks types.ServerStreamConnectionCallbacks) types.ServerStreamConnection {
	return newServerStreamConnection(connection, callbacks)
}

// types.DecodeFilter
// types.StreamConnection
type streamConnection struct {
	protocol      types.Protocol
	connection    types.Connection
	activeStreams map[uint32]*stream
	protocols     types.Protocols
}

// types.StreamConnection
func (conn *streamConnection) Dispatch(buffer types.IoBuffer) {
	conn.protocols.Decode(buffer, conn)
}

func (conn *streamConnection) Protocol() types.Protocol {
	return conn.protocol
}

// types.DecodeFilter
func (conn *streamConnection) OnDecodeHeader(streamId uint32, headers map[string]string) types.FilterStatus {
	if stream, ok := conn.activeStreams[streamId]; ok {
		stream.decoder.DecodeHeaders(headers, false)
	}

	return types.StopIteration
}

func (conn *streamConnection) OnDecodeData(streamId uint32, data types.IoBuffer) types.FilterStatus {
	if stream, ok := conn.activeStreams[streamId]; ok {
		stream.decoder.DecodeData(data, false)
	}

	return types.Continue
}

func (conn *streamConnection) OnDecodeTrailer(streamId uint32, trailers map[string]string) types.FilterStatus {
	if stream, ok := conn.activeStreams[streamId]; ok {
		stream.decoder.DecodeTrailers(trailers)
	}

	return types.Continue
}

func (conn *streamConnection) OnDecodeComplete(streamId uint32, buf types.IoBuffer) {
	if stream, ok := conn.activeStreams[streamId]; ok {
		stream.decoder.DecodeComplete(buf)
	}
}

// types.ClientStreamConnection
type clientStreamConnection struct {
	streamConnection
	streamConnCallbacks types.StreamConnectionCallbacks
}

func newClientStreamConnection(connection types.Connection,
	callbacks types.StreamConnectionCallbacks) types.ClientStreamConnection {

	return &clientStreamConnection{
		streamConnection: streamConnection{
			connection:    connection,
			protocols:     sofarpc.DefaultProtocols(),
			activeStreams: make(map[uint32]*stream),
		},
		streamConnCallbacks: callbacks,
	}
}

func (c *clientStreamConnection) NewStream(streamId uint32, responseDecoder types.StreamDecoder) types.StreamEncoder {
	stream := &stream{
		streamId:   streamId,
		connection: &c.streamConnection,
		decoder:    responseDecoder,
	}

	c.activeStreams[streamId] = stream

	return stream
}

// types.ServerStreamConnection
type serverStreamConnection struct {
	streamConnection
	serverStreamConnCallbacks types.ServerStreamConnectionCallbacks
}

func newServerStreamConnection(connection types.Connection,
	callbacks types.ServerStreamConnectionCallbacks) types.ServerStreamConnection {
	return &serverStreamConnection{
		streamConnection: streamConnection{
			connection:    connection,
			protocols:     sofarpc.DefaultProtocols(),
			activeStreams: make(map[uint32]*stream),
		},
		serverStreamConnCallbacks: callbacks,
	}
}

func (sc *serverStreamConnection) Dispatch(buffer types.IoBuffer) {
	sc.protocols.Decode(buffer, sc)
}

// types.DecodeFilter
func (sc *serverStreamConnection) OnDecodeHeader(streamId uint32, headers map[string]string) types.FilterStatus {
	if streamId == 0 {
		return types.Continue
	}

	sc.onNewStreamDetected(streamId)
	sc.streamConnection.OnDecodeHeader(streamId, headers)

	return types.StopIteration
}

func (sc *serverStreamConnection) OnDecodeData(streamId uint32, data types.IoBuffer) types.FilterStatus {
	if streamId == 0 {
		return types.Continue
	}

	sc.onNewStreamDetected(streamId)
	sc.streamConnection.OnDecodeData(streamId, data)

	return types.StopIteration
}

func (sc *serverStreamConnection) onNewStreamDetected(streamId uint32) {
	if _, ok := sc.activeStreams[streamId]; ok {
		return
	}

	stream := &stream{
		streamId:   streamId,
		connection: &sc.streamConnection,
	}

	stream.decoder = sc.serverStreamConnCallbacks.NewStream(streamId, stream)
	sc.activeStreams[streamId] = stream
}

// types.Stream
// types.StreamEncoder
type stream struct {
	streamId         uint32
	readDisableCount int
	connection       *streamConnection
	decoder          types.StreamDecoder
	streamCbs        []types.StreamCallbacks
}

// ~~ types.Stream
func (s *stream) AddCallbacks(cb types.StreamCallbacks) {
	s.streamCbs = append(s.streamCbs, cb)
}

func (s *stream) RemoveCallbacks(cb types.StreamCallbacks) {
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
// we just send raw request data in first stage
func (s *stream) EncodeHeaders(headers map[string]string, endStream bool) {
	// skip encode headers
}

func (s *stream) EncodeData(data types.IoBuffer, endStream bool) {
	// just send data in current solution
	s.connection.connection.Write(data)
}

func (s *stream) EncodeTrailers(trailers map[string]string) {
	// skip encode trailers
}

func (s *stream) GetStream() types.Stream {
	return s
}
