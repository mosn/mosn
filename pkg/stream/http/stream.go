package http
//
//import (
//	"container/list"
//	"context"
//	"fmt"
//	"io"
//	"net"
//	"net/http"
//	"net/url"
//	"strconv"
//	"strings"
//	"sync/atomic"
//	"time"
//
//	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
//	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
//	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
//	str "gitlab.alipay-inc.com/afe/mosn/pkg/stream"
//	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
//	"golang.org/x/net/http2"
//	"github.com/valyala/fasthttp"
//)
//
//func init() {
//	str.Register(protocol.Http1, &streamConnFactory{})
//}
//
//type streamConnFactory struct{}
//
//func (f *streamConnFactory) CreateClientStream(context context.Context, connection types.ClientConnection,
//	streamConnCallbacks types.StreamConnectionEventListener, connCallbacks types.ConnectionEventListener) types.ClientStreamConnection {
//	return newClientStreamConnection(context, connection, streamConnCallbacks, connCallbacks)
//}
//
//func (f *streamConnFactory) CreateServerStream(context context.Context, connection types.Connection,
//	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
//	return newServerStreamConnection(context, connection, callbacks)
//}
//
//func (f *streamConnFactory) CreateBiDirectStream(context context.Context, connection types.ClientConnection,
//	clientCallbacks types.StreamConnectionEventListener,
//	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {
//	return nil
//}
//
//// types.StreamConnection
//// types.StreamConnectionEventListener
//type streamConnection struct {
//	context context.Context
//
//	protocol      types.Protocol
//	rawConnection net.Conn
//	activeStream  stream //concurrent stream not supported in HTTP/1.X, so list storage for active stream is not required
//	connCallbacks types.ConnectionEventListener
//
//	logger log.Logger
//}
//
//// types.StreamConnection
//func (conn *streamConnection) Dispatch(buffer types.IoBuffer) {}
//
//func (conn *streamConnection) Protocol() types.Protocol {
//	return conn.protocol
//}
//
//func (conn *streamConnection) GoAway() {
//	// todo
//}
//
//func (conn *streamConnection) OnUnderlyingConnectionAboveWriteBufferHighWatermark() {
//	for _, cb := range conn.activeStream.streamCbs {
//		cb.OnAboveWriteBufferHighWatermark()
//	}
//}
//
//func (conn *streamConnection) OnUnderlyingConnectionBelowWriteBufferLowWatermark() {
//	for _, cb := range conn.activeStream.streamCbs {
//		cb.OnBelowWriteBufferLowWatermark()
//	}
//}
//
//// types.ClientStreamConnection
//type clientStreamConnection struct {
//	streamConnection
//	client              *fasthttp.HostClient
//	streamConnCallbacks types.StreamConnectionEventListener
//}
//
//func newClientStreamConnection(context context.Context, connection types.ClientConnection,
//	streamConnCallbacks types.StreamConnectionEventListener,
//	connCallbacks types.ConnectionEventListener) types.ClientStreamConnection {
//
//
//		c := fasthttp.HostClient{
//			Dial: func(addr string) (net.Conn, error) {
//
//			},
//
//			Addr: connection.RemoteAddr().String(),
//
//			DialDualStack: true,
//		}
//		c.dO
//
//	h2Conn, _ := transport.NewClientConn(connection.RawConn())
//
//	return &clientStreamConnection{
//		streamConnection: streamConnection{
//			context:       context,
//			rawConnection: connection.RawConn(),
//			http2Conn:     h2Conn,
//			connCallbacks: connCallbacks,
//		},
//		streamConnCallbacks: streamConnCallbacks,
//	}
//}
//
//func (csc *clientStreamConnection) OnGoAway() {
//	csc.streamConnCallbacks.OnGoAway()
//}
//
//func (csc *clientStreamConnection) NewStream(streamId string, responseDecoder types.StreamDecoder) types.StreamEncoder {
//	stream := &clientStream{
//		stream: stream{
//			context: context.WithValue(csc.context, types.ContextKeyStreamId, streamId),
//			decoder: responseDecoder,
//		},
//		connection: csc,
//	}
//
//	csc.asMutex.Lock()
//	stream.element = csc.activeStreams.PushBack(stream)
//	csc.asMutex.Unlock()
//
//	return stream
//}
//
//// types.ServerStreamConnection
//type serverStreamConnection struct {
//	streamConnection
//	serverStreamConnCallbacks types.ServerStreamConnectionEventListener
//}
//
//func newServerStreamConnection(context context.Context, connection types.Connection,
//	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
//	ssc := &serverStreamConnection{
//		streamConnection: streamConnection{
//			context:       context,
//			rawConnection: connection.RawConn(),
//		},
//		serverStreamConnCallbacks: callbacks,
//	}
//
//	fasthttp.ServeConn(connection.RawConn(), ssc.ServeHTTP)
//
//	return ssc
//}
//
//func (ssc *serverStreamConnection) OnGoAway() {
//	ssc.serverStreamConnCallbacks.OnGoAway()
//}
//
////作为PROXY的STREAM SERVER
//func (ssc *serverStreamConnection) ServeHTTP(ctx *fasthttp.RequestCtx) {
//	//generate stream id using timestamp
//	streamId := "streamID-" + time.Now().String()
//
//	s := &serverStream{
//		stream: stream{
//			context: context.WithValue(ssc.context, types.ContextKeyStreamId, streamId),
//			ctx:     ctx,
//		},
//		connection:       ssc,
//		responseDoneChan: make(chan bool, 1),
//	}
//
//	s.decoder = ssc.serverStreamConnCallbacks.NewStream(streamId, s)
//
//	ssc.activeStream = s.stream
//
//	if atomic.LoadInt32(&s.readDisableCount) <= 0 {
//		s.handleRequest()
//	}
//
//	select {
//	case <-s.responseDoneChan:
//	}
//}
//
//// types.Stream
//// types.StreamEncoder
//type stream struct {
//	context context.Context
//
//	readDisableCount int32
//	ctx              *fasthttp.RequestCtx
//	decoder          types.StreamDecoder
//	streamCbs        []types.StreamEventListener
//}
//
//// types.Stream
//func (s *stream) AddEventListener(streamCb types.StreamEventListener) {
//	s.streamCbs = append(s.streamCbs, streamCb)
//}
//
//func (s *stream) RemoveEventListener(streamCb types.StreamEventListener) {
//	cbIdx := -1
//
//	for i, streamCb := range s.streamCbs {
//		if streamCb == streamCb {
//			cbIdx = i
//			break
//		}
//	}
//
//	if cbIdx > -1 {
//		s.streamCbs = append(s.streamCbs[:cbIdx], s.streamCbs[cbIdx+1:]...)
//	}
//}
//
//func (s *stream) ResetStream(reason types.StreamResetReason) {
//	for _, cb := range s.streamCbs {
//		cb.OnResetStream(reason)
//	}
//}
//
//type clientStream struct {
//	stream
//	connection *clientStreamConnection
//}
//
//// types.StreamEncoder
//func (s *clientStream) EncodeHeaders(headers_ interface{}, endStream bool) error {
//	headers, _ := headers_.(map[string]string)
//
//	if s.request == nil {
//		s.request = new(http.Request)
//		s.request.Method = http.MethodGet
//		s.request.URL, _ = url.Parse(fmt.Sprintf("http://%s/",
//			s.connection.rawConnection.RemoteAddr().String()))
//	}
//
//	if method, ok := headers[types.HeaderMethod]; ok {
//		s.request.Method = method
//		delete(headers, types.HeaderMethod)
//	}
//
//	if host, ok := headers[types.HeaderHost]; ok {
//		s.request.Host = host
//		delete(headers, types.HeaderHost)
//	}
//
//	if path, ok := headers[types.HeaderPath]; ok {
//		s.request.URL, _ = url.Parse(fmt.Sprintf("http://%s%s",
//			s.connection.rawConnection.RemoteAddr().String(), path))
//		delete(headers, types.HeaderPath)
//	}
//
//	s.request.Header = encodeHeader(headers)
//
//	if endStream {
//		s.endStream()
//	}
//
//	return nil
//}
//
//func (s *clientStream) EncodeData(data types.IoBuffer, endStream bool) error {
//	if s.request == nil {
//		s.request = new(http.Request)
//	}
//
//	s.request.Body = &IoBufferReadCloser{
//		buf: data,
//	}
//
//	if endStream {
//		s.endStream()
//	}
//
//	return nil
//}
//
//func (s *clientStream) EncodeTrailers(trailers map[string]string) error {
//	s.request.Trailer = encodeHeader(trailers)
//	s.endStream()
//
//	return nil
//}
//
//func (s *clientStream) endStream() {
//	s.doSend()
//}
//
//func (s *clientStream) ReadDisable(disable bool) {
//	s.connection.logger.Debugf("high watermark on h2 stream client")
//
//	if disable {
//		atomic.AddInt32(&s.readDisableCount, 1)
//	} else {
//		newCount := atomic.AddInt32(&s.readDisableCount, -1)
//
//		if newCount <= 0 {
//			s.handleResponse()
//		}
//	}
//}
//
//func (s *clientStream) doSend() {
//	resp, err := s.connection.http2Conn.RoundTrip(s.request)
//
//	if err != nil {
//		// due to we use golang h2 conn impl, we need to do some adapt to some things observable
//		switch err.(type) {
//		case http2.StreamError:
//			s.ResetStream(types.StreamRemoteReset)
//		case http2.GoAwayError:
//			s.connection.streamConnCallbacks.OnGoAway()
//		case error:
//			// todo: target remote close event
//			if err == io.EOF {
//				s.connection.connCallbacks.OnEvent(types.RemoteClose)
//			} else if err.Error() == "http2: client conn is closed" {
//				// we dont use mosn io impl, so get connection state from golang h2 io read/write loop
//				s.connection.connCallbacks.OnEvent(types.LocalClose)
//			} else if err.Error() == "http2: client conn not usable" {
//				// raise overflow event to let conn pool taking action
//				s.ResetStream(types.StreamOverflow)
//			} else if err.Error() == "http2: Transport received Server's graceful shutdown GOAWAY" {
//				s.connection.streamConnCallbacks.OnGoAway()
//			} else if err.Error() == "http2: Transport received Server's graceful shutdown GOAWAY; some request body already written" {
//				// todo: retry
//			} else if err.Error() == "http2: timeout awaiting response headers" {
//				s.ResetStream(types.StreamConnectionFailed)
//			} else if err.Error() == "net/http: request canceled" {
//				s.ResetStream(types.StreamLocalReset)
//			} else {
//				s.connection.logger.Errorf("Unknown err: %v", err)
//				s.ResetStream(types.StreamConnectionFailed)
//			}
//		}
//	} else {
//		s.response = resp
//
//		if atomic.LoadInt32(&s.readDisableCount) <= 0 {
//			s.handleResponse()
//		}
//	}
//}
//
//func (s *clientStream) handleResponse() {
//	if s.response != nil {
//		s.decoder.OnDecodeHeaders(decodeHeader(s.response.Header), false)
//		buf := &buffer.IoBuffer{}
//		buf.ReadFrom(s.response.Body)
//		s.decoder.OnDecodeData(buf, false)
//		s.decoder.OnDecodeTrailers(decodeHeader(s.response.Trailer))
//
//		s.connection.asMutex.Lock()
//		s.response = nil
//		s.connection.activeStreams.Remove(s.element)
//		s.connection.asMutex.Unlock()
//	}
//}
//
//func (s *clientStream) GetStream() types.Stream {
//	return s
//}
//
//type serverStream struct {
//	stream
//	connection       *serverStreamConnection
//	responseDoneChan chan bool
//}
//
//// types.StreamEncoder
//func (s *serverStream) EncodeHeaders(headers_ interface{}, endStream bool) error {
//	headers, _ := headers_.(map[string]string)
//
//	if s.response == nil {
//		s.response = new(http.Response)
//		s.response.StatusCode = 200
//	}
//
//	s.response.Header = encodeHeader(headers)
//
//	if status := s.response.Header.Get(types.HeaderStatus); status != "" {
//		s.response.StatusCode, _ = strconv.Atoi(status)
//		s.response.Header.Del(types.HeaderStatus)
//	}
//
//	if endStream {
//		s.endStream()
//	}
//	return nil
//}
//
//func (s *serverStream) EncodeData(data types.IoBuffer, endStream bool) error {
//	if s.response == nil {
//		s.response = new(http.Response)
//	}
//
//	s.response.Body = &IoBufferReadCloser{
//		buf: data,
//	}
//
//	if endStream {
//		s.endStream()
//	}
//	return nil
//}
//
//func (s *serverStream) EncodeTrailers(trailers map[string]string) error {
//	s.response.Trailer = encodeHeader(trailers)
//
//	s.endStream()
//	return nil
//}
//
//func (s *serverStream) endStream() {
//	s.doSend()
//	s.responseDoneChan <- true
//
//	s.connection.asMutex.Lock()
//	s.connection.activeStreams.Remove(s.element)
//	s.connection.asMutex.Unlock()
//}
//
//func (s *serverStream) ReadDisable(disable bool) {
//	s.connection.logger.Debugf("high watermark on h2 stream server")
//
//	if disable {
//		atomic.AddInt32(&s.readDisableCount, 1)
//	} else {
//		newCount := atomic.AddInt32(&s.readDisableCount, -1)
//
//		if newCount <= 0 {
//			s.handleRequest()
//		}
//	}
//}
//
//func (s *serverStream) doSend() {
//	for key, values := range s.response.Header {
//		for _, value := range values {
//			s.responseWriter.Header().Add(key, value)
//		}
//	}
//
//	s.responseWriter.WriteHeader(s.response.StatusCode)
//
//	buf := &buffer.IoBuffer{}
//	buf.ReadFrom(s.response.Body)
//	buf.WriteTo(s.responseWriter)
//}
//
//func (s *serverStream) handleRequest() {
//	if s.ctx != nil {
//		s.decoder.OnDecodeHeaders(decodeHeader(s.ctx.Request.Header), false)
//		buf := buffer.NewIoBufferBytes(s.ctx.Request.Body())
//		s.decoder.OnDecodeData(buf, true)
//		//no Trailer in Http/1.x
//	}
//}
//
//func (s *serverStream) GetStream() types.Stream {
//	return s
//}
//
//func encodeHeader(in map[string]string) (out map[string][]string) {
//	out = make(map[string][]string, len(in))
//
//	for k, v := range in {
//		out[k] = strings.Split(v, ",")
//	}
//
//	return
//}
//
//func decodeHeader(in fasthttp.RequestHeader) (out map[string]string) {
//	out = make(map[string]string, in.Len())
//
//	in.VisitAll(func(key, value []byte){
//		// convert to lower case for internal process
//		out[strings.ToLower(string(key))] = string(value)
//	})
//
//	return
//}
//
//// io.ReadCloser
//type IoBufferReadCloser struct {
//	buf types.IoBuffer
//}
//
//func (rc *IoBufferReadCloser) Read(p []byte) (n int, err error) {
//	return rc.buf.Read(p)
//}
//
//func (rc *IoBufferReadCloser) Close() error {
//	rc.buf.Reset()
//	return nil
//}
