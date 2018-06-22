package sofarpc

import (
	"context"
	"sync"
	
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	str "gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type StreamDirection int

const (
	ServerStream StreamDirection = 1
	ClientStream StreamDirection = 0
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
	context         context.Context
	protocol        types.Protocol
	connection      types.Connection
	protocols       types.Protocols
	activeStream    streamMap
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
		activeStream:    newStreamMap(context),
		clientCallbacks: clientCallbacks,
		serverCallbacks: serverCallbacks,
		logger:          log.ByContext(context),
	}
}

// types.StreamConnection
func (conn *streamConnection) Dispatch(buffer types.IoBuffer) {
	conn.protocols.Decode(conn.context, buffer, conn)
}

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

func (conn *streamConnection) NewStream(streamId string, responseDecoder types.StreamDecoder) types.StreamEncoder {
	stream := stream{
		context:    context.WithValue(conn.context, types.ContextKeyStreamId, streamId),
		streamId:   streamId,
		requestId:  streamId,
		direction:  ClientStream,
		connection: conn,
		decoder:    responseDecoder,
	}
	conn.activeStream.Set(streamId, stream)

	return &stream
}

func (conn *streamConnection) OnDecodeHeader(streamId string, headers map[string]string) types.FilterStatus {
	if sofarpc.IsSofaRequest(headers) {
		conn.onNewStreamDetected(streamId, headers)
	}
	endStream := decodeSterilize(streamId, headers)

	if stream, ok := conn.activeStream.Get(streamId); ok {
		stream.decoder.OnDecodeHeaders(headers, endStream)
		if endStream {
			return types.StopIteration
		}
	}

	return types.Continue
}

func (conn *streamConnection) OnDecodeData(streamId string, data types.IoBuffer) types.FilterStatus {
	if stream, ok := conn.activeStream.Get(streamId); ok {
		stream.decoder.OnDecodeData(data, true)

		if stream.direction == ClientStream {
			// for client stream, remove stream on response read
			stream.connection.activeStream.Remove(stream.streamId)
		}
	}

	return types.StopIteration
}

func (conn *streamConnection) OnDecodeTrailer(streamId string, trailers map[string]string) types.FilterStatus {
	// unsupported
	return types.StopIteration
}

// todo, deal with more exception
func (conn *streamConnection) OnDecodeError(err error, header map[string]string) {
	if err == nil {
		return
	}

	// for header decode error, close the connection directly
	if err.Error() == types.UnSupportedProCode {
		conn.connection.Close(types.NoFlush, types.LocalClose)
	}

	if err.Error() == sofarpc.UnKnownCmdcode {
		conn.connection.Close(types.NoFlush, types.LocalClose)
	}

	if err.Error() == sofarpc.UnKnownReqtype {
		conn.connection.Close(types.NoFlush, types.LocalClose)
	}
	
	// for other exception
	if err.Error() == types.CodecException {
		var streamId string
		if v, ok := header[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)]; ok {
			streamId = v
		} else {
			// if no request id found, no reason to send response, so close connection
			conn.connection.Close(types.NoFlush, types.LocalClose)
		}
		
		conn.onNewStreamDetected(streamId,header)
		
		if stream, ok := conn.activeStream.Get(streamId); ok {
			stream.decoder.OnDecodeError(err,header)
		}
	}
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
		context:    context.WithValue(conn.context, types.ContextKeyStreamId, streamId),
		streamId:   streamId,
		requestId:  requestId,
		direction:  ServerStream,
		connection: conn,
	}

	stream.decoder = conn.serverCallbacks.NewStream(streamId, &stream)
	conn.activeStream.Set(streamId, stream)
}

// types.Stream
// types.StreamEncoder
type stream struct {
	context context.Context

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
	var err error
	
	if err, s.encodedHeaders = s.connection.protocols.EncodeHeaders(s.context, s.encodeSterilize(headers)); err != nil {
		return err
	}
	
	//log.StartLogger.Debugf("EncodeHeaders,request id = %s, direction = %d",s.streamId,s.direction)
	
	if endStream {
		s.endStream()
	}

	return nil
}

func (s *stream) EncodeData(data types.IoBuffer, endStream bool) error {
	s.encodedData = data
	
	//log.StartLogger.Debugf("EncodeData,request id = %s, direction = %d",s.streamId,s.direction)
	
	if endStream {
		s.endStream()
	}

	return nil
}

func (s *stream) EncodeTrailers(trailers map[string]string) error {
	s.endStream()

	return nil
}

// Flush stream data
// For server stream, write out response
// For client stream, write out request
func (s *stream) endStream() {
	if s.encodedHeaders != nil {
		log.StartLogger.Debugf("Write, stream id = %s, direction = %d",s.streamId,s.direction)
		
		if stream, ok := s.connection.activeStream.Get(s.streamId); ok {

			if s.encodedData != nil {
			//	log.DefaultLogger.Debugf("[response data1 Response Body is full]",s.encodedHeaders.Bytes(),time.Now().String())
				stream.connection.connection.Write(s.encodedHeaders, s.encodedData)
			} else {
			//	s.connection.logger.Debugf("stream %s response body is void...", s.streamId)
				stream.connection.connection.Write(s.encodedHeaders)
			}
		} else {
			s.connection.logger.Errorf("No stream %s to end", s.streamId)
		}
	} else {
		s.connection.logger.Debugf("Response Headers is void...")
	}

	if s.direction == ServerStream {
		// for a server stream, remove stream on response wrote
		s.connection.activeStream.Remove(s.streamId)
	//	log.StartLogger.Warnf("Remove Request ID = %+v",s.streamId)
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
