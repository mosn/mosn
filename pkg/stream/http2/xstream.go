package http2

import (
	"bufio"
	"context"
	"io"
	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/module/http2"
	"mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/protocol"
	mhttp2 "mosn.io/mosn/pkg/protocol/http2"
	str "mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
	"mosn.io/pkg/buffer"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// types.DecodeFilter
// types.StreamConnection
// types.ClientStreamConnection
// types.ServerStreamConnection
type xStreamConnection struct {
	ctx  context.Context
	conn api.Connection
	cm   *str.ContextManager

	bufChan    chan buffer.IoBuffer
	endRead    chan struct{}
	connClosed chan bool

	br *bufio.Reader

	useStream bool

	protocol api.Protocol
}

func (conn *xStreamConnection) Protocol() types.ProtocolName {
	return protocol.HTTP2
}

func (conn *xStreamConnection) EnableWorkerPool() bool {
	return true
}

func (conn *xStreamConnection) GoAway() {
	// todo
}

type xServerStreamConnection struct {
	xStreamConnection
	mutex   sync.RWMutex
	streams map[uint32]*xServerStream
	sc      *http2.XServerConn
	config  StreamConfig

	serverCallbacks types.ServerStreamConnectionEventListener
}

func newXServerStreamConnection(ctx context.Context, connection api.Connection, serverCallbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {

	sc := &xServerStreamConnection{
		xStreamConnection: xStreamConnection{
			ctx:        ctx,
			conn:       connection,
			bufChan:    make(chan buffer.IoBuffer),
			endRead:    make(chan struct{}),
			connClosed: make(chan bool, 1),

			cm: str.NewContextManager(ctx),
		},
		config: parseStreamConfig(ctx),

		serverCallbacks: serverCallbacks,
	}
	h2sc := http2.NewXServerConn(connection, sc, nil)
	sc.sc = h2sc
	sc.protocol = mhttp2.XServerProto(h2sc)

	if pgc := mosnctx.Get(ctx, types.ContextKeyProxyGeneralConfig); pgc != nil {
		if extendConfig, ok := pgc.(StreamConfig); ok {
			sc.useStream = extendConfig.Http2UseStream
		}
	}

	// init first context
	sc.cm.Next()

	// set not support transfer connection
	sc.conn.SetTransferEventListener(func() bool {
		return false
	})

	sc.streams = make(map[uint32]*xServerStream, 32)

	connection.AddConnectionEventListener(sc)
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(ctx, "new http2 server stream connection, stream config: %v", sc.config)
	}

	return sc
}

func (conn *xServerStreamConnection) OnEvent(event api.ConnectionEvent) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	if event.IsClose() || event.ConnectFailure() {
		for _, stream := range conn.streams {
			stream.ResetStream(types.StreamRemoteReset)
		}
	}
}

// types.StreamConnectionM
func (conn *xServerStreamConnection) Dispatch(buf types.IoBuffer) {
	for {
		_, err := conn.protocol.Decode(nil, buf)
		if err != nil {
			break
		}
	}
}

func (conn *xServerStreamConnection) ActiveStreamsNum() int {
	conn.mutex.RLock()
	defer conn.mutex.Unlock()

	return len(conn.streams)
}

func (conn *xServerStreamConnection) CheckReasonError(connected bool, event api.ConnectionEvent) (types.StreamResetReason, bool) {
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

func (conn *xServerStreamConnection) Reset(reason types.StreamResetReason) {
	conn.mutex.RLock()
	defer conn.mutex.Unlock()

	for _, stream := range conn.streams {
		stream.ResetStream(reason)
	}
}

// decode失败错误怎么处理？
func (conn *xServerStreamConnection) HandleFrame(f http2.Frame, h2s *http2.XStream, data []byte, hasTrailer, endStream bool, err error) {
	ctx := conn.cm.Get()
	defer conn.cm.Next()

	if h2s == nil && data == nil && !hasTrailer && !endStream {
		return
	}

	id := f.Header().StreamID

	var stream *xServerStream
	// header
	if h2s != nil {
		stream, err = conn.onNewStreamDetect(mosnctx.Clone(ctx), h2s, endStream)
		if err != nil {
			conn.handleError(ctx, f, err)
			return
		}
		header := mhttp2.NewReqHeader(h2s.Request)

		scheme := "http"
		if _, ok := conn.conn.RawConn().(*mtls.TLSConn); ok {
			scheme = "https"
		}

		h2s.Request.URL.Scheme = strings.ToLower(scheme)

		variable.SetString(ctx, types.VarScheme, scheme)
		variable.SetString(ctx, types.VarMethod, h2s.Request.Method)
		variable.SetString(ctx, types.VarHost, h2s.Request.Host)
		variable.SetString(ctx, types.VarIstioHeaderHost, h2s.Request.Host) // be consistent with http1
		variable.SetString(ctx, types.VarPath, h2s.Request.URL.Path)
		variable.SetString(ctx, types.VarPathOriginal, h2s.Request.URL.EscapedPath())

		if h2s.Request.URL.RawQuery != "" {
			variable.SetString(ctx, types.VarQueryString, h2s.Request.URL.RawQuery)
		}

		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(stream.ctx, "http2 server header: %d, %+v", id, h2s.Request.Header)
		}

		if endStream {
			stream.receiver.OnReceive(stream.ctx, header, nil, nil)
			return
		}
		stream.header = header
		stream.trailer = &mhttp2.HeaderMap{}
	}

	if stream == nil {
		stream = conn.onStreamRecv(ctx, id, endStream)
		if stream == nil {
			log.Proxy.Errorf(ctx, "http2 server OnStreamRecv error, invaild id = %d", id)
			return
		}
	}

	// data
	if data != nil {
		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(ctx, "http2 server receive data: %d", id)
		}

		if stream.recData == nil {
			if endStream || !conn.useStream {
				stream.recData = buffer.GetIoBuffer(len(data))
			} else {
				stream.recData = buffer.NewPipeBuffer(len(data))
				stream.reqUseStream = true
				variable.Set(stream.ctx, types.VarHttp2RequestUseStream, stream.reqUseStream)
				stream.receiver.OnReceive(stream.ctx, stream.header, stream.recData, stream.trailer)
			}
		}

		if _, err = stream.recData.Write(data); err != nil {
			conn.handleError(ctx, f, http2.StreamError{
				StreamID: id,
				Code:     http2.ErrCodeCancel,
				Cause:    err,
			})
			return
		}
	}

	if hasTrailer {
		stream.trailer.H = stream.h2s.Request.Trailer
		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(stream.ctx, "http2 server trailer: %d, %v", id, stream.h2s.Request.Trailer)
		}
	}

	if endStream {
		if stream.reqUseStream {
			stream.recData.CloseWithError(io.EOF)
		} else {
			if stream.recData == nil {
				stream.recData = buffer.GetIoBuffer(0)
			}
			stream.receiver.OnReceive(stream.ctx, stream.header, stream.recData, stream.trailer)
		}
		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(stream.ctx, "http2 server stream end %d", id)
		}
	}

}

func (conn *xServerStreamConnection) handleError(ctx context.Context, f http2.Frame, err error) {
	conn.sc.HandleError(f, err)
	if err != nil {
		switch err := err.(type) {
		// todo: other error scenes
		case http2.StreamError:
			if err.Code == http2.ErrCodeNo {
				return
			}
			log.Proxy.Errorf(ctx, "Http2 server handleError stream error: %v", err)
			conn.mutex.Lock()
			s := conn.streams[err.StreamID]
			if s != nil {
				delete(conn.streams, err.StreamID)
			}
			conn.mutex.Unlock()
			if s != nil {
				s.ResetStream(types.StreamLocalReset)
			}
		case http2.ConnectionError:
			log.Proxy.Errorf(ctx, "Http2 server handleError conn err: %v", err)
			conn.conn.Close(api.NoFlush, api.OnReadErrClose)
		default:
			log.Proxy.Errorf(ctx, "Http2 server handleError err: %v", err)
			conn.conn.Close(api.NoFlush, api.RemoteClose)
		}
	}
}

func (conn *xServerStreamConnection) onNewStreamDetect(ctx context.Context, h2s *http2.XStream, endStream bool) (*xServerStream, error) {
	stream := &xServerStream{}
	stream.id = h2s.ID()
	stream.ctx = mosnctx.WithValue(ctx, types.ContextKeyStreamID, stream.id)
	stream.sc = conn
	stream.h2s = h2s
	stream.conn = conn.conn

	conn.mutex.Lock()
	conn.streams[stream.id] = stream
	conn.mutex.Unlock()

	var span api.Span
	if trace.IsEnabled() {
		// try build trace span
		tracer := trace.Tracer(protocol.HTTP2)

		if tracer != nil {
			span = tracer.Start(ctx, h2s.Request, time.Now())
		}
	}

	stream.receiver = conn.serverCallbacks.NewStreamDetect(stream.ctx, stream, span)
	return stream, nil
}

func (conn *xServerStreamConnection) onStreamRecv(ctx context.Context, id uint32, endStream bool) *xServerStream {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	if stream, ok := conn.streams[id]; ok {
		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(stream.ctx, "http2 server OnStreamRecv, id = %d", stream.id)
		}
		return stream
	}
	return nil
}

type xServerStream struct {
	stream
	h2s           *http2.XStream
	sc            *xServerStreamConnection
	reqUseStream  bool
	respUseStream bool
}

// types.StreamSender
func (s *xServerStream) AppendHeaders(ctx context.Context, headers api.HeaderMap, endStream bool) error {
	if useStream, err := variable.Get(ctx, types.VarHttp2ResponseUseStream); err == nil {
		if h2UseStream, ok := useStream.(bool); ok {
			s.respUseStream = h2UseStream
		}
	}
	var rsp *http.Response

	var status int

	value, err := variable.GetString(ctx, types.VarHeaderStatus)
	if err != nil || value == "" {
		status = 200
	} else {
		status, _ = strconv.Atoi(value)
	}

	switch header := headers.(type) {
	case *mhttp2.RspHeader:
		rsp = header.Rsp
	case *mhttp2.ReqHeader:
		// indicates the invocation is under hijack scene
		rsp = new(http.Response)
		rsp.StatusCode = status
		rsp.Header = s.h2s.Request.Header
	default:
		rsp = new(http.Response)
		rsp.StatusCode = status
		rsp.Header = mhttp2.EncodeHeader(headers)
	}

	s.h2s.Response = rsp
	s.h2s.UseStream = s.respUseStream

	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(s.ctx, "http2 server ApppendHeaders id = %d, headers = %+v", s.id, rsp.Header)
	}

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *xServerStream) AppendData(context context.Context, data buffer.IoBuffer, endStream bool) error {
	s.h2s.SendData = data
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(s.ctx, "http2 server ApppendData id = %d", s.id)
	}

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *xServerStream) AppendTrailers(context context.Context, trailers api.HeaderMap) error {
	if trailers != nil {
		switch trailer := trailers.(type) {
		case *mhttp2.HeaderMap:
			s.h2s.Trailer = &trailer.H
		default:
			header := mhttp2.EncodeHeader(trailer)
			s.h2s.Trailer = &header
		}
		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(s.ctx, "http2 server ApppendTrailers id = %d, trailer = %+v", s.id, s.h2s.Trailer)
		}
	}
	s.endStream()

	return nil
}

func (s *xServerStream) ResetStream(reason types.StreamResetReason) {
	// on stream reset
	if log.Proxy.GetLogLevel() >= log.WARN {
		log.Proxy.Warnf(s.ctx, "http2 server reset stream id = %d, error = %v", s.id, reason)
	}
	if s.reqUseStream && s.recData != nil {
		s.recData.CloseWithError(io.EOF)
	}

	s.stream.ResetStream(reason)
}

func (s *xServerStream) GetStream() types.Stream {
	return s
}

func (s *xServerStream) endStream() {
	if s.h2s.SendData != nil {
		// Need to reset the 'Content-Length' response header when it's a direct response.
		isDirectResponse, _ := variable.GetString(s.ctx, types.VarProxyIsDirectResponse)
		if isDirectResponse == types.IsDirectResponse {
			s.h2s.Response.Header.Set("Content-Length", strconv.Itoa(s.h2s.SendData.Len()))
		}
	}

	_, err := s.sc.protocol.Encode(s.ctx, s.h2s)

	s.sc.mutex.Lock()
	delete(s.sc.streams, s.id)
	s.sc.mutex.Unlock()

	if err != nil {
		log.Proxy.Errorf(s.ctx, "http2 server SendResponse error :%v", err)
		s.ResetStream(types.StreamLocalReset)
		return
	}
	if s.reqUseStream && s.recData != nil {
		s.recData.CloseWithError(io.EOF)
	}

	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(s.ctx, "http2 server SendResponse id = %d", s.id)
	}
}
