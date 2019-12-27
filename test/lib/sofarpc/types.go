package sofarpc

import (
	"sync"
	"sync/atomic"
	"time"

	"mosn.io/mosn/pkg/types"
)

// Server
type ServeFunc func(types.IoBuffer) *WriteResponseData

type WriteResponseData struct {
	Status      int16
	DataToWrite []byte
}

type ServerStats struct {
	connectionTotal  uint32
	connectionActive uint32
	connectionClosed uint32
	requestTotal     uint32

	mutex         sync.Mutex
	responseTotal map[int16]uint32 //statuscode: count
}

func NewServerStats() *ServerStats {
	return &ServerStats{
		mutex:         sync.Mutex{},
		responseTotal: make(map[int16]uint32),
	}
}

func (s *ServerStats) ActiveConnection() {
	atomic.AddUint32(&s.connectionTotal, 1)
	atomic.AddUint32(&s.connectionActive, 1)
}

func (s *ServerStats) CloseConnection() {
	// subtract x, add ^uint32(x-1)
	// subtract 1 add  ^uint32(0)
	atomic.AddUint32(&s.connectionActive, ^uint32(0))
	atomic.AddUint32(&s.connectionClosed, 1)
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

func (s *ServerStats) ConnectionStats() (uint32, uint32, uint32) {
	return s.connectionTotal, s.connectionActive, s.connectionClosed
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
	Status uint32
	Header map[string]string
	Data   []byte
	Cost   time.Duration
}

type MakeRequestFunc func(id uint64, header map[string]string, body []byte) (types.HeaderMap, types.IoBuffer)

type ResponseVerify func(*Response) bool

type ClientStats struct {
	*ServerStats
	expectedResponse   uint32
	unexpectedResponse uint32
}

func NewClientStats() *ClientStats {
	return &ClientStats{
		ServerStats: NewServerStats(),
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
