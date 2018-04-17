package sofarpc

import (
	"errors"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	str "gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"strconv"
	"sync"
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
	protocol   types.Protocol
	connection types.Connection
	activeStream
	protocols       types.Protocols
	clientCallbacks types.StreamConnectionEventListener
	serverCallbacks types.ServerStreamConnectionEventListener
}

var as activeStream

func newStreamConnection(connection types.Connection, clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {

	return &streamConnection{
		connection:      connection,
		protocols:       sofarpc.DefaultProtocols(),
		activeStream:    as.init(),
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
		direction:  0, //out
		connection: conn,
		decoder:    responseDecoder,
	}

	conn.activeStream.addStream(streamId, stream)
	return stream
}

// types.DecodeFilter Called by serverStreamConnection
func (conn *streamConnection) OnDecodeHeader(streamId string, headers map[string]string) types.FilterStatus {

	if sofarpc.IsSofaRequest(headers) {
		conn.onNewStreamDetected(streamId)
	}

	if v, ok := headers[sofarpc.SofaPropertyHeader("requestid")]; ok {
		headers[types.HeaderStreamID] = v
	}

	if v, ok := headers[sofarpc.SofaPropertyHeader("timeout")]; ok {
		headers[types.HeaderTryTimeout] = v
	}

	if v, ok := headers[sofarpc.SofaPropertyHeader("globaltimeout")]; ok {
		headers[types.HeaderGlobalTimeout] = v
	}

	if stream, ok := conn.activeStream.getStream(streamId); ok {
		stream.decoder.OnDecodeHeaders(headers, false) //Call Back Proxy-Level's OnDecodeHeaders
	} else if v, ok := headers[types.HeaderException]; ok && v == types.MosnExceptionCodeC {
		// codec exception may happen
		conn.onNewStreamDetected(streamId)
		stream, _ = conn.activeStream.getStream(streamId)
		stream.decoder.OnDecodeHeaders(headers, true)
	}

	return types.Continue
}

func (conn *streamConnection) OnDecodeData(streamId string, data types.IoBuffer) types.FilterStatus {

	if stream, ok := conn.activeStream.getStream(streamId); ok {
		stream.decoder.OnDecodeData(data, true)

		if stream.direction == 0 {
			stream.connection.activeStream.delStream(stream.streamId)
		}
	}

	return types.StopIteration
}

func (conn *streamConnection) OnDecodeTrailer(streamId string, trailers map[string]string) types.FilterStatus {

	if stream, ok := conn.activeStream.getStream(streamId); ok {
		stream.decoder.OnDecodeTrailers(trailers)
	}

	return types.StopIteration
}

func (conn *streamConnection) onNewStreamDetected(streamId string) {

	if _, ok := conn.activeStream.getStream(streamId); ok {
		return
	}

	stream := &stream{
		streamId:   streamId,
		direction:  1, //in
		connection: conn,
	}

	stream.decoder = conn.serverCallbacks.NewStream(streamId, stream)
	conn.activeStream.addStream(streamId, stream)
}

// types.Stream
// types.StreamEncoder
type stream struct {
	streamId         string
	direction        int // 0: out, 1: in
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
	if headerMaps, ok := headers.(map[string]string); ok {

		// remove proxy header before codec encode
		if _, ok := headerMaps[types.HeaderStreamID]; ok {
			delete(headerMaps, types.HeaderStreamID)
		}

		if _, ok := headerMaps[types.HeaderGlobalTimeout]; ok {
			delete(headerMaps, types.HeaderGlobalTimeout)
		}

		if _, ok := headerMaps[types.HeaderTryTimeout]; ok {
			delete(headerMaps, types.HeaderTryTimeout)
		}

		if status, ok := headerMaps[types.HeaderStatus]; ok {

			delete(headerMaps, types.HeaderStatus)
			statusCode, _ := strconv.Atoi(status)

			//todo: handle proxy hijack reply on exception @boqin
			if statusCode != types.SuccessCode {

				var respHeaders interface{}
				var err error

				//Build Router Unavailable Response Msg
				switch statusCode {

				case types.RouterUnavailableCode, types.NoHealthUpstreamCode, types.UpstreamOverFlowCode:
					//No available path
					respHeaders, err = sofarpc.BuildSofaRespMsg(headerMaps, sofarpc.RESPONSE_STATUS_CLIENT_SEND_ERROR)
				case types.CodecExceptionCode:
					//Decode or Encode Error
					respHeaders, err = sofarpc.BuildSofaRespMsg(headerMaps, sofarpc.RESPONSE_STATUS_CODEC_EXCEPTION)
				case types.DeserialExceptionCode:
					//Hessian Exception
					respHeaders, err = sofarpc.BuildSofaRespMsg(headerMaps, sofarpc.RESPONSE_STATUS_SERVER_DESERIAL_EXCEPTION)
				case types.TimeoutExceptionCode:
					//Response Timeout
					respHeaders, err = sofarpc.BuildSofaRespMsg(headerMaps, sofarpc.RESPONSE_STATUS_TIMEOUT)

				default:
					respHeaders, err = sofarpc.BuildSofaRespMsg(headerMaps, sofarpc.RESPONSE_STATUS_UNKNOWN)
				}

				if err == nil {
					switch respHeaders.(type) {
					case *sofarpc.BoltResponseCommand:
						headers = respHeaders.(*sofarpc.BoltResponseCommand)
					case *sofarpc.BoltV2ResponseCommand:
						headers = respHeaders.(*sofarpc.BoltV2ResponseCommand)
					default:
						headers = headerMaps
					}
				} else {
					log.DefaultLogger.Println(err.Error())
					headers = headerMaps
				}

			} else {

				headers = headerMaps
			}

		} else {
			headers = headerMaps
		}
	}
	s.streamId, s.encodedHeaders = s.connection.protocols.EncodeHeaders(headers)

	// Exception occurs when encoding headers
	if s.encodedHeaders == nil {
		return errors.New(s.streamId)
	}
	s.connection.activeStream.addStream(s.streamId, s)

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
		stream, _ := s.connection.activeStream.getStream(s.streamId)
		stream.connection.connection.Write(s.encodedHeaders)

		if s.encodedData != nil {
			stream.connection.connection.Write(s.encodedData)
		} else {
			log.DefaultLogger.Println("Response Body is void...")
		}
	} else {
		log.DefaultLogger.Println("Response Headers is void...")
	}
	if s.direction == 1 {
		s.connection.activeStream.delStream(s.streamId)
	}
}

func (s *stream) GetStream() types.Stream {
	return s
}

type activeStream struct {
	activeStreams map[string]*stream
	asMutex       *sync.RWMutex
}

func (as *activeStream) init() activeStream {
	return activeStream{
		activeStreams: make(map[string]*stream),
		asMutex:       new(sync.RWMutex),
	}
}

func (as *activeStream) getStream(streamId string) (*stream, bool) {
	as.asMutex.RLock()
	defer as.asMutex.RUnlock()
	s, ok := as.activeStreams[streamId]
	return s, ok
}

func (as *activeStream) addStream(streamId string, s *stream) {
	as.asMutex.Lock()
	defer as.asMutex.Unlock()
	as.activeStreams[streamId] = s
}

func (as *activeStream) delStream(streamId string) {
	as.asMutex.Lock()
	defer as.asMutex.Unlock()
	delete(as.activeStreams, streamId)
}
