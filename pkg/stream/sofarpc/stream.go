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
	"github.com/alipay/sofa-mosn/pkg/protocol"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"errors"
	"sync/atomic"
	"strconv"
)

// StreamDirection represent the stream's direction
type StreamDirection int

// ServerStream = 1
// ClientStream = 0
const (
	ServerStream StreamDirection = 1
	ClientStream StreamDirection = 0
)

var (
	ErrNotSofarpcCmd = errors.New("not sofarpc command")
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
	ctx  context.Context
	conn types.Connection

	codecEngine types.ProtocolEngine

	clientCallbacks types.StreamConnectionEventListener
	serverCallbacks types.ServerStreamConnectionEventListener

	streams      streamMap // client conn fields
	currStreamID uint32

	logger log.Logger
}

func newStreamConnection(ctx context.Context, connection types.Connection, clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {

	sc := &streamConnection{
		ctx:             ctx,
		conn:            connection,
		codecEngine:     sofarpc.Engine(),
		clientCallbacks: clientCallbacks,
		serverCallbacks: serverCallbacks,

		logger: log.ByContext(ctx),
	}

	if sc.clientCallbacks != nil {
		sc.streams = newStreamMap(ctx)
	}

	return sc
}

// types.StreamConnection
func (conn *streamConnection) Dispatch(buf types.IoBuffer) {
	conn.codecEngine.Process(conn.ctx, buf, conn.handleCommand)
}

func (conn *streamConnection) Protocol() types.Protocol {
	return protocol.SofaRPC
}

func (conn *streamConnection) GoAway() {
	// todo
}

func (conn *streamConnection) NewStream(ctx context.Context, receiver types.StreamReceiver) types.StreamSender {
	buffers := sofaBuffersByContext(ctx)

	stream := &buffers.client

	stream.ID = atomic.AddUint32(&conn.currStreamID, 1)
	stream.ctx = context.WithValue(ctx, types.ContextKeyStreamID, stream.ID)
	stream.direction = ClientStream
	stream.sc = conn
	stream.receiver = receiver

	conn.streams.Set(stream.ID, stream)
	return stream
}

func (conn *streamConnection) handleCommand(ctx context.Context, model interface{}, err error) {
	if err != nil {
		conn.handleError(model, err)
		return
	}

	var stream *stream

	cmd, ok := model.(sofarpc.SofaRpcCmd)

	if !ok {
		conn.handleError(model, ErrNotSofarpcCmd)
		return
	}

	switch cmd.CommandType() {
	case sofarpc.REQUEST, sofarpc.REQUEST_ONEWAY:
		stream = conn.onNewStreamDetect(cmd)

		//req := cmd.(*sofarpc.BoltRequest)
		//conn.logger.Debugf("Decode conn %d streamID %d sofarpc command, length %d", conn.conn.ID(), stream.ID, 22+int(req.ClassLen)+int(req.HeaderLen)+req.ContentLen)
	case sofarpc.RESPONSE:
		stream = conn.onStreamRecv(cmd)

		//resp := cmd.(*sofarpc.BoltResponse)
		//conn.logger.Debugf("Decode conn %d streamID %d sofarpc command, length %d", conn.conn.ID(), stream.ID, 20+int(resp.ClassLen)+int(resp.HeaderLen)+resp.ContentLen)
	}

	// header, data notify
	if stream != nil {
		header := cmd.Header()
		data := cmd.Data()

		if header != nil {
			// special logic: set protocol code into header map
			//header[sofarpc.HeaderProtocolCode] = strconv.FormatUint(uint64(cmd.GetProtocol()), 10)
			stream.receiver.OnReceiveHeaders(stream.ctx, cmd, data == nil)
		}

		if data != nil {
			stream.receiver.OnReceiveData(stream.ctx, buffer.NewIoBufferBytes(data), true)
		}
	}
}

func (conn *streamConnection) handleError(cmd interface{}, err error) {

	switch err {
	case rpc.ErrUnrecognizedCode, sofarpc.ErrUnKnownCmdType, sofarpc.ErrUnKnownCmdCode, ErrNotSofarpcCmd:
		conn.logger.Errorf("error occurs while proceeding codec logic: %s", err.Error())
		//protocol decode error, close the connection directly
		conn.conn.Close(types.NoFlush, types.LocalClose)
	case types.ErrCodecException, types.ErrDeserializeException:
		if cmd, ok := cmd.(sofarpc.SofaRpcCmd); ok {
			if reqID := cmd.RequestID(); reqID > 0 {

				// TODO: to see some error handling if is necessary to passed to proxy level, or just handle it at stream level
				var stream *stream
				switch cmd.CommandType() {
				case sofarpc.REQUEST, sofarpc.REQUEST_ONEWAY:
					stream = conn.onNewStreamDetect(cmd)
				case sofarpc.RESPONSE:
					stream, _ = conn.streams.Get(reqID)
				}

				// valid sofarpc cmd with positive requestID, send exception response in this case
				if stream != nil {
					if stream.direction == ClientStream {
						// for client stream, remove stream on response read
						conn.streams.Remove(stream.ID)
					}
					stream.receiver.OnDecodeError(stream.ctx, err, cmd)
				}
				return
			}
		}
		// if no request id found, no reason to send response, so close connection
		conn.conn.Close(types.NoFlush, types.LocalClose)
	}
}

func (conn *streamConnection) onNewStreamDetect(cmd sofarpc.SofaRpcCmd) *stream {
	buffers := sofaBuffersByContext(conn.ctx)
	stream := &buffers.server
	stream.ID = cmd.RequestID()
	stream.ctx = context.WithValue(conn.ctx, types.ContextKeyStreamID, stream.ID)
	stream.direction = ServerStream
	stream.sc = conn

	conn.logger.Debugf("new stream detect, id = %d", stream.ID)

	stream.receiver = conn.serverCallbacks.NewStreamDetect(stream.ctx, stream)
	return stream
}

func (conn *streamConnection) onStreamRecv(cmd sofarpc.SofaRpcCmd) *stream {
	requestID := cmd.RequestID()

	// for client stream, remove stream on response read
	if stream, ok := conn.streams.Get(requestID); ok {
		conn.streams.Remove(requestID)
		return stream
	}
	return nil
}

// types.Stream
// types.StreamSender
type stream struct {
	ctx context.Context
	sc  *streamConnection

	ID        uint32
	direction StreamDirection // 0: out, 1: in

	receiver  types.StreamReceiver
	streamCbs []types.StreamEventListener

	encodedData types.IoBuffer // for current impl, need removed
	sendCmd     sofarpc.SofaRpcCmd
	sendBuf     types.IoBuffer
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
	s.sc.conn.SetReadDisable(disable)
}

func (s *stream) BufferLimit() uint32 {
	return s.sc.conn.BufferLimit()
}

// types.StreamSender
func (s *stream) AppendHeaders(ctx context.Context, headers types.HeaderMap, endStream bool) error {
	cmd, ok := headers.(sofarpc.SofaRpcCmd)

	if !ok {
		return ErrNotSofarpcCmd
	}

	var err error

	switch s.direction {
	case ClientStream:
		// copy origin request from downstream
		s.sendCmd, err = sofarpc.Clone(s.ctx, cmd)
	case ServerStream:
		switch cmd.CommandType() {
		case sofarpc.RESPONSE:
			// copy origin response from upstream
			s.sendCmd, err = sofarpc.Clone(s.ctx, cmd)
		case sofarpc.REQUEST, sofarpc.REQUEST_ONEWAY:
			// the command type is request, indicates the invocation is under hijack scene
			s.sendCmd, err = s.buildHijackResp(cmd)
		}
	}

	s.sc.logger.Debugf("AppendHeaders,request id = %d, direction = %d", s.ID, s.direction)

	if endStream {
		s.endStream()
	}

	return err
}

func (s *stream) buildHijackResp(request sofarpc.SofaRpcCmd) (sofarpc.SofaRpcCmd, error) {
	if status, ok := request.Get(types.HeaderStatus); ok {
		request.Del(types.HeaderStatus)
		statusCode, _ := strconv.Atoi(status)

		if statusCode != types.SuccessCode {

			//Build Router Unavailable Response Msg
			switch statusCode {
			case types.RouterUnavailableCode, types.NoHealthUpstreamCode, types.UpstreamOverFlowCode:
				//No available path
				return sofarpc.NewResponse(s.ctx, request, sofarpc.RESPONSE_STATUS_CLIENT_SEND_ERROR)
			case types.CodecExceptionCode:
				//Decode or Encode Error
				return sofarpc.NewResponse(s.ctx, request, sofarpc.RESPONSE_STATUS_CODEC_EXCEPTION)
			case types.DeserialExceptionCode:
				//Hessian Exception
				return sofarpc.NewResponse(s.ctx, request, sofarpc.RESPONSE_STATUS_SERVER_DESERIAL_EXCEPTION)
			case types.TimeoutExceptionCode:
				//Response Timeout
				return sofarpc.NewResponse(s.ctx, request, sofarpc.RESPONSE_STATUS_TIMEOUT)
			default:
				return sofarpc.NewResponse(s.ctx, request, sofarpc.RESPONSE_STATUS_UNKNOWN)
			}
		}
	}

	return request, types.ErrNoErrorCodeForHijack
}

func (s *stream) AppendData(context context.Context, data types.IoBuffer, endStream bool) error {
	//if s.sendCmd != nil {
	//	// TODO: may affect buffer reuse
	//	s.sendCmd.SetData(data.Bytes())
	//}

	s.encodedData = data

	log.DefaultLogger.Infof("AppendData,request id = %d, direction = %d", s.ID, s.direction)

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *stream) AppendTrailers(context context.Context, trailers types.HeaderMap) error {
	s.endStream()

	return nil
}

// Flush stream data
// For server stream, write out response
// For client stream, write out request
func (s *stream) endStream() {
	if s.sendCmd != nil {

		// replace requestID
		s.sendCmd.SetRequestID(s.ID)

		// TODO: replaced with EncodeTo, and pre-alloc send buf
		buf, err := s.sc.codecEngine.Encode(s.ctx, s.sendCmd)
		if err != nil {
			s.sc.logger.Errorf("encode error:%s", err.Error())
			s.ResetStream(types.StreamLocalReset)
			return
		}

		if s.encodedData != nil {
			s.sc.conn.Write(buf, s.encodedData)
		} else {
			s.sc.conn.Write(buf)
		}

	}
}

func (s *stream) GetStream() types.Stream {
	return s
}

type streamMap struct {
	smap map[uint32]*stream
	mux  sync.RWMutex
}

func newStreamMap(context context.Context) streamMap {
	smap := make(map[uint32]*stream, 32)

	return streamMap{
		smap: smap,
	}
}

func (m *streamMap) Has(streamID uint32) bool {
	m.mux.RLock()
	if _, ok := m.smap[streamID]; ok {
		m.mux.RUnlock()
		return true
	}
	m.mux.RUnlock()
	return false
}

func (m *streamMap) Get(streamID uint32) (s *stream, ok bool) {
	m.mux.RLock()
	s, ok = m.smap[streamID]
	m.mux.RUnlock()
	return
}

func (m *streamMap) Remove(streamID uint32) {
	m.mux.Lock()
	delete(m.smap, streamID)
	m.mux.Unlock()

}

func (m *streamMap) Set(streamID uint32, s *stream) {
	m.mux.Lock()
	m.smap[streamID] = s
	m.mux.Unlock()
}
