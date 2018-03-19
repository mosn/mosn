package sofarpc

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"net/http"
	"golang.org/x/net/http2"
)

type streamConnection struct {
	protocol      types.Protocol
	connection    types.Connection
	http2Conn     http2.ClientConn
	activeStreams map[uint32]*stream
}

// types.Stream
// types.StreamEncoder
type stream struct {
	streamId         uint32
	readDisableCount int
	request          *http.Request
	response         *http.Response
	connection       *streamConnection
	responseDecoder  types.StreamDecoder
	streamCbs        []types.StreamCallbacks
}

// types.Stream
func (s *stream) AddCallbacks(streamCb types.StreamCallbacks) {

}

func (s *stream) RemoveCallbacks(streamCb types.StreamCallbacks) {

}

func (s *stream) ResetStream(reason types.StreamResetReason) {

}

func (s *stream) ReadDisable(disable bool) {

}

func (s *stream) BufferLimit() uint32 {
	return 0
}

// types.StreamEncoder
func (s *stream) EncodeHeaders(headers map[string][]string, endStream bool) {
	if s.request == nil {
		s.request = new(http.Request)
		s.request.Method = "GET"
	}

	s.request.Header = headers

	if endStream {
		s.endStream()
	}
}

func (s *stream) EncodeData(data types.IoBuffer, endStream bool) {
	if s.request == nil {
		s.request = new(http.Request)
	}

	s.request.Body = data

	if endStream {
		s.endStream()
	}
}

func (s *stream) EncodeTrailers(trailers map[string][]string) {
	s.request.Trailer = trailers
}

func (s *stream) endStream() {
	s.doRequest()
}

func (s *stream) doRequest() {
	s.connection.http2Conn.RoundTrip(s.request)
}

func (s *stream) GetStream() types.Stream {
	return s
}
