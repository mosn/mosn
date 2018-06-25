package http

import (
	"container/list"
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	str "gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"github.com/valyala/fasthttp"
	"sync"
)

func init() {
	str.Register(protocol.Http1, &streamConnFactory{})
}

type streamConnFactory struct{}

func (f *streamConnFactory) CreateClientStream(context context.Context, connection types.ClientConnection,
	streamConnCallbacks types.StreamConnectionEventListener, connCallbacks types.ConnectionEventListener) types.ClientStreamConnection {
	return nil
}

func (f *streamConnFactory) CreateServerStream(context context.Context, connection types.Connection,
	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	return newServerStreamConnection(context, connection, callbacks)
}

func (f *streamConnFactory) CreateBiDirectStream(context context.Context, connection types.ClientConnection,
	clientCallbacks types.StreamConnectionEventListener,
	serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection {
	return nil
}

// types.StreamConnection
// types.StreamConnectionEventListener
type streamConnection struct {
	context context.Context

	protocol      types.Protocol
	rawConnection net.Conn
	activeStream  *stream //concurrent stream not supported in HTTP/1.X, so list storage for active stream is not required
	connCallbacks types.ConnectionEventListener

	logger log.Logger
}

// types.StreamConnection
func (conn *streamConnection) Dispatch(buffer types.IoBuffer) {}

func (conn *streamConnection) Protocol() types.Protocol {
	return conn.protocol
}

func (conn *streamConnection) GoAway() {
	// todo
}

func (conn *streamConnection) OnUnderlyingConnectionAboveWriteBufferHighWatermark() {
	for _, cb := range conn.activeStream.streamCbs {
		cb.OnAboveWriteBufferHighWatermark()
	}
}

func (conn *streamConnection) OnUnderlyingConnectionBelowWriteBufferLowWatermark() {
	for _, cb := range conn.activeStream.streamCbs {
		cb.OnBelowWriteBufferLowWatermark()
	}
}

// since we use fasthttp client, which already implements the conn-pool feature and manage the connnections itself
// there is no need to do the connection management. so http/1.x stream only wrap the progress for constructing request/response
// and have no aware of the connection it would use

// types.ClientStreamConnection
type clientStreamWrapper struct {
	context context.Context

	client *fasthttp.HostClient

	activeStreams *list.List
	asMutex       sync.Mutex

	connCallbacks       types.ConnectionEventListener
	streamConnCallbacks types.StreamConnectionEventListener
}

func newClientStreamWrapper(context context.Context, client *fasthttp.HostClient,
	streamConnCallbacks types.StreamConnectionEventListener,
	connCallbacks types.ConnectionEventListener) types.ClientStreamConnection {

	return &clientStreamWrapper{
		context:             context,
		client:              client,
		activeStreams:       list.New(),
		connCallbacks:       connCallbacks,
		streamConnCallbacks: streamConnCallbacks,
	}
}

// types.StreamConnection
func (csw *clientStreamWrapper) Dispatch(buffer types.IoBuffer) {}

func (csw *clientStreamWrapper) Protocol() types.Protocol {
	return protocol.Http1
}

func (csw *clientStreamWrapper) GoAway() {
	// todo
}

func (csw *clientStreamWrapper) OnUnderlyingConnectionAboveWriteBufferHighWatermark() {
	for as := csw.activeStreams.Front(); as != nil; as = as.Next() {
		for _, cb := range as.Value.(stream).streamCbs {
			cb.OnAboveWriteBufferHighWatermark()
		}
	}
}

func (csw *clientStreamWrapper) OnUnderlyingConnectionBelowWriteBufferLowWatermark() {
	for as := csw.activeStreams.Front(); as != nil; as = as.Next() {
		for _, cb := range as.Value.(stream).streamCbs {
			cb.OnAboveWriteBufferHighWatermark()
		}
	}
}

func (csw *clientStreamWrapper) OnGoAway() {
	csw.streamConnCallbacks.OnGoAway()
}

func (csw *clientStreamWrapper) NewStream(streamId string, responseDecoder types.StreamDecoder) types.StreamEncoder {
	stream := &clientStream{
		stream: stream{
			context: context.WithValue(csw.context, types.ContextKeyStreamId, streamId),
			decoder: responseDecoder,
		},
		wrapper: csw,
	}

	csw.asMutex.Lock()
	stream.element = csw.activeStreams.PushBack(stream)
	csw.asMutex.Unlock()

	return stream
}

// types.ServerStreamConnection
type serverStreamConnection struct {
	streamConnection
	serverStreamConnCallbacks types.ServerStreamConnectionEventListener
}

func newServerStreamConnection(context context.Context, connection types.Connection,
	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
	ssc := &serverStreamConnection{
		streamConnection: streamConnection{
			context:       context,
			rawConnection: connection.RawConn(),
		},
		serverStreamConnCallbacks: callbacks,
	}

	fasthttp.ServeConn(connection.RawConn(), ssc.ServeHTTP)

	return ssc
}

func (ssc *serverStreamConnection) OnGoAway() {
	ssc.serverStreamConnCallbacks.OnGoAway()
}

//作为PROXY的STREAM SERVER
func (ssc *serverStreamConnection) ServeHTTP(ctx *fasthttp.RequestCtx) {
	//generate stream id using timestamp
	streamId := "streamID-" + time.Now().String()

	s := &serverStream{
		stream: stream{
			context: context.WithValue(ssc.context, types.ContextKeyStreamId, streamId),
		},
		ctx:              ctx,
		connection:       ssc,
		responseDoneChan: make(chan bool, 1),
	}

	s.decoder = ssc.serverStreamConnCallbacks.NewStream(streamId, s)

	ssc.activeStream = &s.stream

	if atomic.LoadInt32(&s.readDisableCount) <= 0 {
		s.handleRequest()
	}

	select {
	case <-s.responseDoneChan:
		s.ctx = nil
		ssc.activeStream = nil
	}
}

// types.Stream
// types.StreamEncoder
type stream struct {
	context context.Context

	readDisableCount int32

	decoder   types.StreamDecoder
	streamCbs []types.StreamEventListener
}

// types.Stream
func (s *stream) AddEventListener(streamCb types.StreamEventListener) {
	s.streamCbs = append(s.streamCbs, streamCb)
}

func (s *stream) RemoveEventListener(streamCb types.StreamEventListener) {
	cbIdx := -1

	for i, streamCb := range s.streamCbs {
		if streamCb == streamCb {
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

type clientStream struct {
	stream

	// NOTICE: fasthttp ctx and its member not allowed holding by others after request handle finished
	request  *fasthttp.Request
	response *fasthttp.Response

	element *list.Element
	wrapper *clientStreamWrapper
}

// types.StreamEncoder
func (s *clientStream) EncodeHeaders(headers_ interface{}, endStream bool) error {
	headers, _ := headers_.(map[string]string)

	if s.request == nil {
		s.request = fasthttp.AcquireRequest()
		s.request.Header.SetMethod(http.MethodGet)
		s.request.SetRequestURI(fmt.Sprintf("http://%s/", s.wrapper.client.Addr))
	}

	if method, ok := headers[types.HeaderMethod]; ok {
		s.request.Header.SetMethod(method)
		delete(headers, types.HeaderMethod)
	}

	if host, ok := headers[types.HeaderHost]; ok {
		s.request.SetHost(host)
		delete(headers, types.HeaderHost)
	}

	if path, ok := headers[types.HeaderPath]; ok {
		s.request.SetRequestURI(fmt.Sprintf("http://%s%s", s.wrapper.client.Addr, path))
		delete(headers, types.HeaderPath)
	}

	encodeReqHeader(s.request, headers)

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *clientStream) EncodeData(data types.IoBuffer, endStream bool) error {
	if s.request == nil {
		s.request = fasthttp.AcquireRequest()
	}

	s.request.SetBody(data.Bytes())

	if endStream {
		s.endStream()
	}

	return nil
}

func (s *clientStream) EncodeTrailers(trailers map[string]string) error {
	s.endStream()

	return nil
}

func (s *clientStream) endStream() {
	s.doSend()
}

func (s *clientStream) ReadDisable(disable bool) {
	//s.connection.logger.Debugf("high watermark on h2 stream client")

	if disable {
		atomic.AddInt32(&s.readDisableCount, 1)
	} else {
		newCount := atomic.AddInt32(&s.readDisableCount, -1)

		if newCount <= 0 {
			s.handleResponse()
		}
	}
}

func (s *clientStream) doSend() {
	if (s.response == nil) {
		s.response = fasthttp.AcquireResponse()
	}

	err := s.wrapper.client.Do(s.request, s.response)

	if err != nil {
		//TODO error handle
		fmt.Printf("http1 client stream send error: %+v", err)
		s.wrapper.connCallbacks.OnEvent(types.RemoteClose)
	} else {

		if atomic.LoadInt32(&s.readDisableCount) <= 0 {
			s.handleResponse()
		}
	}
}

func (s *clientStream) handleResponse() {
	if s.response != nil {
		s.decoder.OnDecodeHeaders(decodeRespHeader(s.response.Header), false)
		buf := buffer.NewIoBufferBytes(s.response.Body())
		s.decoder.OnDecodeData(buf, true)

		s.wrapper.asMutex.Lock()
		s.response = nil
		s.wrapper.activeStreams.Remove(s.element)
		s.wrapper.asMutex.Unlock()
	}
}

func (s *clientStream) GetStream() types.Stream {
	return s
}

type serverStream struct {
	stream

	// NOTICE: fasthttp ctx and its member not allowed holding by others after request handle finished
	ctx *fasthttp.RequestCtx

	connection       *serverStreamConnection
	responseDoneChan chan bool
}

// types.StreamEncoder
func (s *serverStream) EncodeHeaders(headers_ interface{}, endStream bool) error {
	headers, _ := headers_.(map[string]string)

	if status, ok := headers[types.HeaderStatus]; ok {
		statusCode, _ := strconv.Atoi(string(status))
		s.ctx.SetStatusCode(statusCode)
		delete(headers, types.HeaderStatus)
	}

	encodeRespHeader(&s.ctx.Response, headers)

	if endStream {
		s.endStream()
	}
	return nil
}

func (s *serverStream) EncodeData(data types.IoBuffer, endStream bool) error {

	s.ctx.SetBody(data.Bytes())

	if endStream {
		s.endStream()
	}
	return nil
}

func (s *serverStream) EncodeTrailers(trailers map[string]string) error {
	s.endStream()
	return nil
}

func (s *serverStream) endStream() {
	s.doSend()
	s.responseDoneChan <- true

	s.connection.activeStream = nil
}

func (s *serverStream) ReadDisable(disable bool) {
	s.connection.logger.Debugf("high watermark on h2 stream server")

	if disable {
		atomic.AddInt32(&s.readDisableCount, 1)
	} else {
		newCount := atomic.AddInt32(&s.readDisableCount, -1)

		if newCount <= 0 {
			s.handleRequest()
		}
	}
}

func (s *serverStream) doSend() {
	// fasthttp will automaticlly flush headers and body
}

func (s *serverStream) handleRequest() {
	if s.ctx != nil {
		s.decoder.OnDecodeHeaders(decodeReqHeader(s.ctx.Request.Header), false)

		//remove detect
		if s.connection.activeStream != nil {
			buf := buffer.NewIoBufferBytes(s.ctx.Request.Body())
			s.decoder.OnDecodeData(buf, true)
			//no Trailer in Http/1.x
		}
	}
}

func (s *serverStream) GetStream() types.Stream {
	return s
}

func encodeReqHeader(req *fasthttp.Request, in map[string]string) {
	for k, v := range in {
		req.Header.Set(k, v)
	}
}

func encodeRespHeader(resp *fasthttp.Response, in map[string]string) {
	for k, v := range in {
		resp.Header.Set(k, v)
	}
}

func decodeReqHeader(in fasthttp.RequestHeader) (out map[string]string) {
	out = make(map[string]string, in.Len())

	in.VisitAll(func(key, value []byte) {
		// convert to lower case for internal process
		out[strings.ToLower(string(key))] = string(value)
	})

	return
}

func decodeRespHeader(in fasthttp.ResponseHeader) (out map[string]string) {
	out = make(map[string]string, in.Len())

	in.VisitAll(func(key, value []byte) {
		// convert to lower case for internal process
		out[strings.ToLower(string(key))] = string(value)
	})

	return
}

// io.ReadCloser
type IoBufferReadCloser struct {
	buf types.IoBuffer
}

func (rc *IoBufferReadCloser) Read(p []byte) (n int, err error) {
	return rc.buf.Read(p)
}

func (rc *IoBufferReadCloser) Close() error {
	rc.buf.Reset()
	return nil
}
