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

	"errors"
	"strconv"
	"sync/atomic"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	mosnctx "github.com/alipay/sofa-mosn/pkg/context"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/trace"
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

var (
	directionText = map[StreamDirection]string{
		ClientStream: "request",
		ServerStream: "response",
	}

	ErrNotSofarpcCmd      = errors.New("not sofarpc command")
	ErrNotResponseBuilder = errors.New("no response builder")
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

func (f *streamConnFactory) ProtocolMatch(prot string, magic []byte) error {
	return str.FAILED
}

// types.DecodeFilter
// types.StreamConnection
// types.ClientStreamConnection
// types.ServerStreamConnection
type streamConnection struct {
	ctx                                 context.Context
	conn                                types.Connection
	contextManager                      *str.ContextManager
	mutex                               sync.RWMutex
	currStreamID                        uint64
	streams                             map[uint64]*stream // client conn fields
	codecEngine                         types.ProtocolEngine
	streamConnectionEventListener       types.StreamConnectionEventListener
	serverStreamConnectionEventListener types.ServerStreamConnectionEventListener
}

func newStreamConnection(ctx context.Context, connection types.Connection, clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {

	sc := &streamConnection{
		ctx:                                 ctx,
		conn:                                connection,
		codecEngine:                         sofarpc.Engine(),
		streamConnectionEventListener:       clientCallbacks,
		serverStreamConnectionEventListener: serverCallbacks,

		contextManager: str.NewContextManager(ctx),
	}

	// init first context
	sc.contextManager.Next()

	if sc.streamConnectionEventListener != nil {
		sc.streams = make(map[uint64]*stream, 32)
	}

	// set support transfer connection
	sc.conn.SetTransferEventListener(func() bool {
		return true
	})

	return sc
}

// types.StreamConnection
func (conn *streamConnection) Dispatch(buf types.IoBuffer) {
	for {
		// 1. pre alloc stream-level ctx with bufferCtx
		ctx := conn.contextManager.Get()

		// 2. decode process
		// TODO: maybe pass sub protocol type
		cmd, err := conn.codecEngine.Decode(ctx, buf)
		// No enough data
		if cmd == nil && err == nil {
			break
		}

		// Do handle staff. Error would also be passed to this function.
		conn.handleCommand(ctx, cmd, err)
		if err != nil {
			break
		}

		conn.contextManager.Next()
	}
}

func (conn *streamConnection) Protocol() types.Protocol {
	return protocol.SofaRPC
}

func (conn *streamConnection) GoAway() {
	// unsupported
}

func (conn *streamConnection) ActiveStreamsNum() int {
	conn.mutex.RLock()
	defer conn.mutex.RUnlock()

	return len(conn.streams)
}

func (conn *streamConnection) Reset(reason types.StreamResetReason) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	for _, stream := range conn.streams {
		stream.ResetStream(reason)
	}
}

func (conn *streamConnection) NewStream(ctx context.Context, receiver types.StreamReceiveListener) types.StreamSender {
	buffers := sofaBuffersByContext(ctx)
	stream := &buffers.client

	//stream := &stream{}

	stream.id = atomic.AddUint64(&conn.currStreamID, 1)
	stream.ctx = mosnctx.WithValue(ctx, types.ContextKeyStreamID, stream.id)
	stream.direction = ClientStream
	stream.sc = conn
	stream.receiver = receiver

	if stream.receiver != nil {
		conn.mutex.Lock()
		conn.streams[stream.id] = stream
		conn.mutex.Unlock()
	}

	return stream
}

func (conn *streamConnection) handleCommand(ctx context.Context, model interface{}, err error) {
	if err != nil {
		conn.handleError(ctx, model, err)
		return
	}

	cmd, ok := model.(sofarpc.SofaRpcCmd)

	if !ok {
		conn.handleError(ctx, model, ErrNotSofarpcCmd)
		return
	}

	stream := conn.processStream(ctx, cmd)

	// header, data notify
	if stream != nil {
		timeoutInt := cmd.GetTimeout()
		timeout := strconv.Itoa(timeoutInt) // timeout, ms
		cmd.Set(types.HeaderGlobalTimeout, timeout)

		stream.receiver.OnReceive(stream.ctx, cmd, cmd.Data(), nil)
	}
}

func (conn *streamConnection) handleError(ctx context.Context, cmd interface{}, err error) {
	switch err {
	case rpc.ErrUnrecognizedCode, sofarpc.ErrUnKnownCmdType, sofarpc.ErrUnKnownCmdCode, ErrNotSofarpcCmd:
		log.DefaultLogger.Errorf("[stream] [sofarpc] error occurs while proceeding codec logic: %v. close connection", err)
		//protocol decode error, close the connection directly
		conn.conn.Close(types.NoFlush, types.LocalClose)
	case types.ErrCodecException, types.ErrDeserializeException:
		if cmd, ok := cmd.(sofarpc.SofaRpcCmd); ok {
			if reqID := cmd.RequestID(); reqID > 0 {

				// TODO: to see some error handling if is necessary to passed to proxy level, or just handle it at stream level
				stream := conn.processStream(ctx, cmd)

				// valid sofarpc cmd with positive requestID, send exception response in this case
				if stream != nil {
					stream.receiver.OnDecodeError(stream.ctx, err, cmd)
				}
				return
			}
		}
		// if no request id found, no reason to send response, so close connection
		conn.conn.Close(types.NoFlush, types.LocalClose)
	}
}

func (conn *streamConnection) processStream(ctx context.Context, cmd sofarpc.SofaRpcCmd) *stream {
	switch cmd.CommandType() {
	case sofarpc.REQUEST, sofarpc.REQUEST_ONEWAY:
		var span types.Span
		if trace.IsTracingEnabled() {
			// try build trace span
			span = conn.codecEngine.BuildSpan(ctx, cmd)
		}
		return conn.onNewStreamDetect(ctx, cmd, span)
	case sofarpc.RESPONSE:
		return conn.onStreamRecv(ctx, cmd)
	}
	return nil
}

func (conn *streamConnection) onNewStreamDetect(ctx context.Context, cmd sofarpc.SofaRpcCmd, span types.Span) *stream {
	buffers := sofaBuffersByContext(ctx)
	stream := &buffers.server

	//stream := &stream{}
	stream.id = cmd.RequestID()
	stream.ctx = mosnctx.WithValue(ctx, types.ContextKeyStreamID, stream.id)
	stream.ctx = mosnctx.WithValue(ctx, types.ContextSubProtocol, cmd.ProtocolCode())
	stream.ctx = conn.contextManager.InjectTrace(stream.ctx, span)
	stream.direction = ServerStream
	stream.sc = conn

	if log.Proxy.GetLogLevel() >= log.INFO {
		log.Proxy.Infof(stream.ctx, "[stream] [sofarpc] new stream detect, requestId = %v", stream.id)
	}

	sender := stream
	if cmd.CommandType() == sofarpc.REQUEST_ONEWAY {
		sender = nil
	}
	stream.receiver = conn.serverStreamConnectionEventListener.NewStreamDetect(stream.ctx, sender, span)

	return stream
}

func (conn *streamConnection) onStreamRecv(ctx context.Context, cmd sofarpc.SofaRpcCmd) *stream {
	requestID := cmd.RequestID()

	// for client stream, remove stream on response read
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if stream, ok := conn.streams[requestID]; ok {
		delete(conn.streams, requestID)

		// transmit buffer ctx
		buffer.TransmitBufferPoolContext(stream.ctx, ctx)

		if log.Proxy.GetLogLevel() >= log.INFO {
			log.Proxy.Infof(stream.ctx, "[stream] [sofarpc] receive response, requestId = %v", stream.id)
		}
		return stream
	}

	return nil
}

// types.Stream
// types.StreamSender
type stream struct {
	str.BaseStream

	ctx context.Context
	sc  *streamConnection

	id        uint64
	direction StreamDirection // 0: out, 1: in
	receiver  types.StreamReceiveListener
	sendCmd   sofarpc.SofaRpcCmd
	sendBuf   types.IoBuffer
}

// ~~ types.Stream
func (s *stream) ID() uint64 {
	return s.id
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
		// use origin request from downstream
		s.sendCmd = cmd
	case ServerStream:
		switch cmd.CommandType() {
		case sofarpc.RESPONSE:
			// use origin response from upstream
			s.sendCmd = cmd
		case sofarpc.REQUEST, sofarpc.REQUEST_ONEWAY:
			// the command type is request, indicates the invocation is under hijack scene
			s.sendCmd, err = s.buildHijackResp(cmd)
		}
	}

	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(s.ctx, "[stream] [sofarpc] %s appendHeaders, requestId = %d", directionText[s.direction], s.id)
	}

	if endStream {
		s.endStream()
	}

	return err
}

func (s *stream) buildHijackResp(request sofarpc.SofaRpcCmd) (sofarpc.SofaRpcCmd, error) {
	if status, ok := request.Get(types.HeaderStatus); ok {
		request.Del(types.HeaderStatus)
		statusCode, _ := strconv.Atoi(status)

		hijackResp := sofarpc.NewResponse(request.ProtocolCode(), sofarpc.MappingFromHttpStatus(statusCode))
		if hijackResp != nil {
			return hijackResp, nil
		}
		return nil, ErrNotResponseBuilder
	}

	return nil, types.ErrNoStatusCodeForHijack
}

func (s *stream) AppendData(context context.Context, data types.IoBuffer, endStream bool) error {
	if s.sendCmd != nil {
		// TODO: may affect buffer reuse
		s.sendCmd.SetData(data)
	}

	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(s.ctx, "[stream] [sofarpc] %s appendData, requestId = %d", directionText[s.direction], s.id)
	}

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
	defer func() {
		if s.direction == ServerStream {
			s.DestroyStream()
		}
	}()

	if s.sendCmd != nil {
		// replace requestID
		s.sendCmd.SetRequestID(s.id)
		// remove the inject header
		s.sendCmd.Del(types.HeaderGlobalTimeout)

		// TODO: replaced with EncodeTo, and pre-alloc send buf
		buf, err := s.sc.codecEngine.Encode(s.ctx, s.sendCmd)
		if err != nil {
			log.Proxy.Errorf(s.ctx, "[stream] [sofarpc] %s encode error:%s", directionText[s.direction], err.Error())
			s.ResetStream(types.StreamLocalReset)
			return
		}

		if dataBuf := s.sendCmd.Data(); dataBuf != nil {
			s.sc.conn.Write(buf, dataBuf)
		} else {
			s.sc.conn.Write(buf)
		}

		// log
		if log.Proxy.GetLogLevel() >= log.INFO {
			log.Proxy.Infof(s.ctx, "[stream] [sofarpc] send %s, requestId = %v", directionText[s.direction], s.id)
		}
	}
}

func (s *stream) GetStream() types.Stream {
	return s
}
