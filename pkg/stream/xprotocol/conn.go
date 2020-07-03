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
	"sync/atomic"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/buffer"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
)

// types.DecodeFilter
// types.StreamConnection
// types.ClientStreamConnection
// types.ServerStreamConnection
type streamConn struct {
	ctx        context.Context
	netConn    api.Connection
	ctxManager *stream.ContextManager

	engine   *xprotocol.XEngine // xprotocol fields
	protocol xprotocol.XProtocol

	serverCallbacks types.ServerStreamConnectionEventListener // server side fields

	clientMutex     sync.RWMutex // client side fields
	clientStreamId  uint64
	clientStreams   map[uint64]*xStream
	clientCallbacks types.StreamConnectionEventListener
}

func newStreamConnection(ctx context.Context, conn api.Connection, clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {

	sc := &streamConn{
		ctx:        ctx,
		netConn:    conn,
		ctxManager: stream.NewContextManager(ctx),

		serverCallbacks: serverCallbacks,
		clientCallbacks: clientCallbacks,
	}

	// 1. init first context
	sc.ctxManager.Next()

	// 2. prepare protocols
	subProtocol := mosnctx.Get(ctx, types.ContextSubProtocol).(string)
	subProtocols := strings.Split(subProtocol, ",")
	// 2.1 exact protocol, get directly
	// 2.2 multi protocol, setup engine for further match
	if len(subProtocols) == 1 {
		proto := xprotocol.GetProtocol(types.ProtocolName(subProtocol))
		if proto == nil {
			log.Proxy.Errorf(ctx, "[stream] [xprotocol] no such protocol: %s", subProtocol)
			return nil
		}
		sc.protocol = proto
	} else {
		engine, err := xprotocol.NewXEngine(subProtocols)
		if err != nil {
			log.Proxy.Errorf(ctx, "[stream] [xprotocol] create XEngine failed: %s", err)
			return nil
		}
		sc.engine = engine
	}

	// client
	if sc.clientCallbacks != nil {
		// default client concurrency capacity: 8
		sc.clientStreams = make(map[uint64]*xStream, 8)
		// TODO: keepalive trigger
	}

	// set support transfer connection
	sc.netConn.SetTransferEventListener(func() bool {
		return true
	})

	return sc
}

func (sc *streamConn) CheckReasonError(connected bool, event api.ConnectionEvent) (types.StreamResetReason, bool) {
	reason := types.StreamConnectionSuccessed
	if event.IsClose() || event.ConnectFailure() {
		reason = types.StreamConnectionFailed
		if connected {
			reason = types.StreamConnectionTermination
		}
		return reason, false

	}

	return reason, true
}

// types.StreamConnection
func (sc *streamConn) Dispatch(buf types.IoBuffer) {
	// match if multi protocol used
	if sc.protocol == nil {
		// 1. try to get ALPN negotiated protocol
		if conn, ok := sc.netConn.RawConn().(*mtls.TLSConn); ok {
			name := conn.ConnectionState().NegotiatedProtocol
			if name != "" {
				proto := xprotocol.GetProtocol(types.ProtocolName(name))
				if proto == nil {
					log.Proxy.Errorf(sc.ctx, "[stream] [xprotocol] negotiated protocol not exists: %s", name)
					// close conn
					sc.netConn.Close(api.NoFlush, api.OnReadErrClose)
					return
				}
				sc.protocol = proto
			}
		}
		// 2. tls ALPN prortocol not exists, try to recognize data
		if sc.protocol == nil {
			proto, result := sc.engine.Match(sc.ctx, buf)
			switch result {
			case types.MatchSuccess:
				sc.protocol = proto.(xprotocol.XProtocol)
			case types.MatchFailed:
				// print error info
				size := buf.Len()
				if size > 10 {
					size = 10
				}
				log.Proxy.Errorf(sc.ctx, "[stream] [xprotocol] engine match failed for magic :%v", buf.Bytes()[:size])
				// close conn
				sc.netConn.Close(api.NoFlush, api.OnReadErrClose)
				return
			case types.MatchAgain:
				// do nothing and return, wait for more data
				return
			}
		}
	}

	// decode frames
	for {
		// 1. get stream-level ctx with bufferCtx
		streamCtx := sc.ctxManager.Get()

		// 2. decode process
		frame, err := sc.protocol.Decode(streamCtx, buf)

		// 2.1 no enough data, break loop
		if frame == nil && err == nil {
			return
		}

		// 2.2 handle error
		if err != nil {
			// print error info
			size := buf.Len()
			if size > 10 {
				size = 10
			}
			log.Proxy.Errorf(sc.ctx, "[stream] [xprotocol] conn %d, %v decode error: %v, buf data: %v", sc.netConn.ID(), sc.netConn.RemoteAddr(), err, buf.Bytes()[:size])

			sc.handleError(streamCtx, frame, err)
			return
		}

		// 2.3 handle frame
		if frame != nil {
			xframe, ok := frame.(xprotocol.XFrame)
			if !ok {
				log.Proxy.Errorf(sc.ctx, "[stream] [xprotocol] conn %d, %v frame type not match : %T", sc.netConn.ID(), sc.netConn.RemoteAddr(), frame)
				return
			}
			sc.handleFrame(streamCtx, xframe)
		}

		// 2.4 prepare next
		sc.ctxManager.Next()
	}
}

func (sc *streamConn) Protocol() types.ProtocolName {
	return protocol.Xprotocol
}

func (sc *streamConn) GoAway() {
	// unsupported
	// TODO: client-side conn pool go away
}

func (sc *streamConn) ActiveStreamsNum() int {
	sc.clientMutex.RLock()
	defer sc.clientMutex.RUnlock()

	return len(sc.clientStreams)
}

func (sc *streamConn) Reset(reason types.StreamResetReason) {
	sc.clientMutex.Lock()
	defer sc.clientMutex.Unlock()

	for _, stream := range sc.clientStreams {
		stream.connReset = true
		stream.ResetStream(reason)
	}
}

func (sc *streamConn) NewStream(ctx context.Context, receiver types.StreamReceiveListener) types.StreamSender {
	clientStream := sc.newClientStream(ctx)

	if receiver != nil {
		clientStream.receiver = receiver

		sc.clientMutex.Lock()
		sc.clientStreams[clientStream.id] = clientStream
		sc.clientMutex.Unlock()
	}

	return clientStream
}

func (sc *streamConn) handleError(ctx context.Context, frame interface{}, err error) {
	// valid request frame with positive requestID, send exception response in this case
	if frame != nil {
		if xframe, ok := frame.(xprotocol.XFrame); ok && (xframe.GetStreamType() == xprotocol.Request) {
			if requestId := xframe.GetRequestId(); requestId > 0 {

				// TODO: to see some error handling if is necessary to passed to proxy level, or just handle it at stream level
				stream := sc.newServerStream(ctx, xframe)
				stream.receiver = sc.serverCallbacks.NewStreamDetect(stream.ctx, stream, nil)
				stream.receiver.OnDecodeError(stream.ctx, err, xframe.GetHeader())
				return
			}
		}
	}

	//protocol decode error, close the connection directly
	addr := sc.netConn.RemoteAddr()
	log.Proxy.Alertf(sc.ctx, types.ErrorKeyCodec, "error occurs while proceeding codec logic: %v. close connection, remote addr: %v", err, addr)
	sc.netConn.Close(api.NoFlush, api.LocalClose)
}

func (sc *streamConn) handleFrame(ctx context.Context, frame xprotocol.XFrame) {
	switch frame.GetStreamType() {
	case xprotocol.Request:
		sc.handleRequest(ctx, frame, false)
	case xprotocol.RequestOneWay:
		sc.handleRequest(ctx, frame, true)
	case xprotocol.Response:
		sc.handleResponse(ctx, frame)
	}
}

func (sc *streamConn) handleRequest(ctx context.Context, frame xprotocol.XFrame, oneway bool) {
	// 1. heartbeat process
	if frame.IsHeartbeatFrame() {
		hbAck := sc.protocol.Reply(frame)
		hbAckData, err := sc.protocol.Encode(ctx, hbAck)
		if err != nil {
			sc.handleError(ctx, frame, err)
			return
		}
		// TODO: confirm if need write goroutine to avoid invoke write in read goroutine
		sc.netConn.Write(hbAckData)
		return
	}

	// 2. goaway process
	if predicate, ok := frame.(xprotocol.GoAwayPredicate); ok && predicate.IsGoAwayFrame() && sc.clientCallbacks != nil {
		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(ctx, "[stream] [xprotocol] goaway received, requestId = %v", frame.GetRequestId())
		}
		sc.clientCallbacks.OnGoAway()
		return
	}

	// 3. create server stream
	serverStream := sc.newServerStream(ctx, frame)

	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(ctx, "[stream] [xprotocol] new stream detect, requestId = %v", serverStream.id)
	}

	// 4. tracer support
	var span types.Span
	if trace.IsEnabled() {
		// try build trace span
		tracer := trace.Tracer(protocol.Xprotocol)
		if tracer != nil {
			span = tracer.Start(ctx, frame, time.Now())
		}
		serverStream.ctx = sc.ctxManager.InjectTrace(serverStream.ctx, span)
	}

	// 5. inject service info
	if aware, ok := frame.(xprotocol.ServiceAware); ok {
		serviceName := aware.GetServiceName()
		methodName := aware.GetMethodName()

		frame.GetHeader().Set(types.HeaderRPCService, serviceName)
		frame.GetHeader().Set(types.HeaderRPCMethod, methodName)

		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(ctx, "[stream] [xprotocol] frame service aware, requestId = %v, serviceName = %v , methodName = %v", serverStream.id, serviceName, methodName)
		}
	}

	// 6. receiver callback
	var sender types.StreamSender
	if !oneway {
		sender = serverStream
	}
	serverStream.receiver = sc.serverCallbacks.NewStreamDetect(serverStream.ctx, sender, span)
	serverStream.receiver.OnReceive(serverStream.ctx, frame.GetHeader(), frame.GetData(), nil)

}

func (sc *streamConn) handleResponse(ctx context.Context, frame xprotocol.XFrame) {
	requestId := frame.GetRequestId()

	// for client stream, remove stream on response read
	sc.clientMutex.Lock()
	defer sc.clientMutex.Unlock()

	if clientStream, ok := sc.clientStreams[requestId]; ok {
		delete(sc.clientStreams, requestId)

		// transmit buffer ctx
		buffer.TransmitBufferPoolContext(clientStream.ctx, ctx)

		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(clientStream.ctx, "[stream] [xprotocol] receive response, requestId = %v", requestId)
		}

		clientStream.receiver.OnReceive(clientStream.ctx, frame.GetHeader(), frame.GetData(), nil)
	}
}

func (sc *streamConn) newServerStream(ctx context.Context, frame xprotocol.XFrame) *xStream {
	//serverStream := &xStream{}

	buffers := streamBuffersByContext(ctx)
	serverStream := &buffers.serverStream

	serverStream.id = frame.GetRequestId()
	serverStream.direction = stream.ServerStream
	serverStream.ctx = mosnctx.WithValue(ctx, types.ContextKeyStreamID, serverStream.id)
	serverStream.ctx = mosnctx.WithValue(ctx, types.ContextSubProtocol, string(sc.protocol.Name()))
	serverStream.sc = sc

	return serverStream
}

func (sc *streamConn) newClientStream(ctx context.Context) *xStream {
	//clientStream := &xStream{}

	buffers := streamBuffersByContext(ctx)
	clientStream := &buffers.clientStream

	clientStream.id = atomic.AddUint64(&sc.clientStreamId, 1)
	clientStream.direction = stream.ClientStream
	clientStream.ctx = mosnctx.WithValue(ctx, types.ContextKeyStreamID, clientStream.id)
	clientStream.ctx = mosnctx.WithValue(ctx, types.ContextSubProtocol, string(sc.protocol.Name()))
	clientStream.sc = sc

	return clientStream
}
