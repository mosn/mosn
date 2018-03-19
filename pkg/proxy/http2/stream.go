package sofarpc

import (
	"io"
	"net"
	"net/http"
	"golang.org/x/net/http2"
	"container/list"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/proxy"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"strings"
)

func init() {
	proxy.Register(protocol.Http2, &streamConnFactory{})
}

type streamConnFactory struct{}

func (f *streamConnFactory) CreateClientStream(connection types.ClientConnection,
	streamConnCallbacks types.StreamConnectionCallbacks, connCallbacks types.ConnectionCallbacks) types.ClientStreamConnection {
	return newClientStreamConnection(connection, streamConnCallbacks, connCallbacks)
}

func (f *streamConnFactory) CreateServerStream(connection types.ServerConnection,
	callbacks types.ServerStreamConnectionCallbacks) types.ServerStreamConnection {
	return newServerStreamConnection(connection, callbacks)
}

var transport http2.Transport
var server http2.Server

// types.StreamConnection
// types.StreamConnectionCallbacks
type streamConnection struct {
	protocol      types.Protocol
	rawConnection net.Conn
	http2Conn     *http2.ClientConn
	activeStreams *list.List
	connCallbacks types.ConnectionCallbacks
}

// types.StreamConnection
func (conn *streamConnection) Dispatch(buffer types.IoBuffer) {}

func (conn *streamConnection) Protocol() types.Protocol {
	return conn.protocol
}

type clientStreamConnection struct {
	streamConnection
	streamConnCallbacks types.StreamConnectionCallbacks
}

func newClientStreamConnection(connection types.ClientConnection,
	streamConnCallbacks types.StreamConnectionCallbacks,
	connCallbacks types.ConnectionCallbacks) types.ClientStreamConnection {

	h2Conn, _ := transport.NewClientConn(connection.RawConn())

	return &clientStreamConnection{
		streamConnection: streamConnection{
			rawConnection: connection.RawConn(),
			http2Conn:     h2Conn,
			activeStreams: list.New(),
			connCallbacks: connCallbacks,
		},
		streamConnCallbacks: streamConnCallbacks,
	}
}

func (csc *clientStreamConnection) OnGoAway() {
	csc.streamConnCallbacks.OnGoAway()
}

func (csc *clientStreamConnection) NewStream(streamId uint32, responseDecoder types.StreamDecoder) types.StreamEncoder {
	stream := &clientStream{
		stream: stream{
			streamId:        streamId,
			responseDecoder: responseDecoder,
		},
		connection: csc,
	}

	ele := csc.activeStreams.PushBack(stream)
	stream.element = ele

	return stream
}

type serverStreamConnection struct {
	streamConnection
	serverStreamConnCallbacks types.ServerStreamConnectionCallbacks
}

func newServerStreamConnection(connection types.ServerConnection,
	callbacks types.ServerStreamConnectionCallbacks) types.ServerStreamConnection {
	ssc := &serverStreamConnection{
		streamConnection: streamConnection{
			rawConnection: connection.RawConn(),
			activeStreams: list.New(),
		},
		serverStreamConnCallbacks: callbacks,
	}

	server.ServeConn(connection.RawConn(), &http2.ServeConnOpts{
		Handler: ssc,
	})

	return ssc
}

func (ssc *serverStreamConnection) OnGoAway() {
	ssc.serverStreamConnCallbacks.OnGoAway()
}

func (ssc *serverStreamConnection) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	stream := &serverStream{
		stream: stream{
			request: request,
		},
		connection: ssc,
	}

	stream.responseDecoder = ssc.serverStreamConnCallbacks.NewStream(0, stream)

	ele := ssc.activeStreams.PushBack(stream)
	stream.element = ele
}

// types.Stream
// types.StreamEncoder
type stream struct {
	streamId         uint32
	readDisableCount int
	request          *http.Request
	response         *http.Response
	responseDecoder  types.StreamDecoder
	element          *list.Element
	streamCbs        []types.StreamCallbacks
}

// types.Stream
func (s *stream) AddCallbacks(streamCb types.StreamCallbacks) {
	s.streamCbs = append(s.streamCbs, streamCb)
}

func (s *stream) RemoveCallbacks(streamCb types.StreamCallbacks) {
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

func (s *stream) ReadDisable(disable bool) {
	// todo
}

func (s *stream) BufferLimit() uint32 {
	// todo
	return 0
}

func (s *stream) endStream() {
	s.doSend()
}

func (s *stream) doSend() {}

func (s *stream) GetStream() types.Stream {
	return s
}

type clientStream struct {
	stream
	request    *http.Request
	connection *clientStreamConnection
}

func (s *clientStream) doSend() {
	go func() {
		resp, err := s.connection.http2Conn.RoundTrip(s.request)

		// due to we use golang h2 conn impl, we need to do some adapt to some things observable
		switch err.(type) {
		case http2.StreamError:
			s.ResetStream(types.StreamRemoteReset)
		case http2.GoAwayError:
			s.connection.streamConnCallbacks.OnGoAway()
		case error:
			// todo: target remote close event
			if err == io.EOF {
				s.connection.connCallbacks.OnEvent(types.RemoteClose)
			} else if err.Error() == "http2: client conn is closed" {
				// we dont use mosn io impl, so get connection state from golang h2 io read/write loop
				s.connection.connCallbacks.OnEvent(types.LocalClose)
			} else if err.Error() == "http2: client conn not usable" {
				// raise overflow event to let conn pool taking action
				s.ResetStream(types.StreamOverflow)
			} else if err.Error() == "http2: Transport received Server's graceful shutdown GOAWAY" {
				s.connection.streamConnCallbacks.OnGoAway()
			} else if err.Error() == "http2: Transport received Server's graceful shutdown GOAWAY; some request body already written" {
				// todo: retry
			} else if err.Error() == "http2: timeout awaiting response headers" {
				s.ResetStream(types.StreamConnectionFailed)
			} else if err.Error() == "net/http: request canceled" {
				s.ResetStream(types.StreamLocalReset)
			} else {
				s.ResetStream(types.StreamConnectionFailed)
			}
		default:
			s.responseDecoder.DecodeHeaders(decodeHeader(resp.Header), false)

			buf := &buffer.IoBuffer{}
			// todo
			buf.ReadFrom(resp.Body)

			s.responseDecoder.DecodeData(buf, false)
			s.responseDecoder.DecodeTrailers(decodeHeader(resp.Trailer))
		}
	}()
}

// types.StreamEncoder
func (s *clientStream) EncodeHeaders(headers map[string]string, endStream bool) {
	if s.request == nil {
		s.request = new(http.Request)
		s.request.Method = "GET"
	}

	s.request.Header = encodeHeader(headers)

	if endStream {
		s.endStream()
	}
}

func (s *clientStream) EncodeData(data types.IoBuffer, endStream bool) {
	if s.request == nil {
		s.request = new(http.Request)
	}

	s.request.Body = &IoBufferReadCloser{
		buf: data,
	}

	if endStream {
		s.endStream()
	}
}

func (s *clientStream) EncodeTrailers(trailers map[string]string) {
	s.request.Trailer = encodeHeader(trailers)
	s.endStream()
}

type serverStream struct {
	stream
	response       *http.Response
	connection     *serverStreamConnection
	responseWriter http.ResponseWriter
}

func (s *serverStream) doSend() {
	for key, values := range s.response.Header {
		for _, value := range values {
			s.responseWriter.Header().Add(key, value)
		}
	}

	s.responseWriter.WriteHeader(s.response.StatusCode)

	buf := &buffer.IoBuffer{}
	// todo
	buf.ReadFrom(s.response.Body)

	// todo
	s.responseWriter.Write(buf.Bytes())
	buf.Reset()
}

// types.StreamEncoder
func (s *serverStream) EncodeHeaders(headers map[string]string, endStream bool) {
	if s.response == nil {
		s.response = new(http.Response)
		s.response.StatusCode = 200
	}

	s.response.Header = encodeHeader(headers)

	if endStream {
		s.endStream()
	}
}

func (s *serverStream) EncodeData(data types.IoBuffer, endStream bool) {
	if s.response == nil {
		s.response = new(http.Response)
	}

	s.response.Body = &IoBufferReadCloser{
		buf: data,
	}

	if endStream {
		s.endStream()
	}
}

func (s *serverStream) EncodeTrailers(trailers map[string]string) {
	s.response.Trailer = encodeHeader(trailers)

	s.endStream()
}

func encodeHeader(in map[string]string) (out map[string][]string) {
	out = make(map[string][]string)

	for k, v := range in {
		out[k] = strings.Split(v, ",")
	}

	return
}

func decodeHeader(in map[string][]string) (out map[string]string) {
	out = make(map[string]string)

	for k, v := range in {
		out[k] = strings.Join(v, ",")
	}

	return
}
