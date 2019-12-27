package http

import (
	"sync"
	"sync/atomic"
	"time"

	mosnhttp "mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/types"
)

type ServeFunc func(srv *MockServer)

type ConnStats struct {
	connectionTotal  uint32
	connectionActive uint32
	connectionClosed uint32
}

func (s *ConnStats) ActiveConnection() {
	atomic.AddUint32(&s.connectionTotal, 1)
	atomic.AddUint32(&s.connectionActive, 1)
}

func (s *ConnStats) CloseConnection() {
	// subtract x, add ^uint32(x-1)
	atomic.AddUint32(&s.connectionActive, ^uint32(0))
	atomic.AddUint32(&s.connectionClosed, 1)
}

func (s *ConnStats) ConnectionStats() (uint32, uint32, uint32) {
	return s.connectionTotal, s.connectionActive, s.connectionClosed
}

type ServerStats struct {
	requestTotal  uint32
	mutex         sync.Mutex
	responseTotal map[int16]uint32 //statuscode: count
	*ConnStats
}

func NewServerStats(cs *ConnStats) *ServerStats {
	return &ServerStats{
		mutex:         sync.Mutex{},
		responseTotal: make(map[int16]uint32),
		ConnStats:     cs,
	}
}

func (s *ServerStats) Response(status int16) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := s.responseTotal[status]; !ok {
		s.responseTotal[status] = 0
	}
	s.responseTotal[status]++
}

func (s *ServerStats) Request() {
	atomic.AddUint32(&s.requestTotal, 1)
}

func (s *ServerStats) RequestStats() uint32 {
	return s.requestTotal
}

func (s *ServerStats) ResponseStats() map[int16]uint32 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	m := make(map[int16]uint32, len(s.responseTotal))
	for k, v := range s.responseTotal {
		m[k] = v
	}
	return m
}

// Client
type Response struct {
	Header mosnhttp.ResponseHeader
	Data   []byte
	Cost   time.Duration
}

type MakeRequestFunc func(method string, header map[string]string, body []byte) (types.HeaderMap, types.IoBuffer)

type ResponseVerify func(*Response) bool

type ClientStats struct {
	*ServerStats
	expectedResponse   uint32
	unexpectedResponse uint32
}

func NewClientStats() *ClientStats {
	return &ClientStats{
		ServerStats: NewServerStats(&ConnStats{}),
	}
}

func (c *ClientStats) Response(expected bool) {
	if expected {
		atomic.AddUint32(&c.expectedResponse, 1)
	} else {
		atomic.AddUint32(&c.unexpectedResponse, 1)
	}
}

func (c *ClientStats) ExpectedResponse() (uint32, uint32) {
	return c.expectedResponse, c.unexpectedResponse
}
