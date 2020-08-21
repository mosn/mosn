package types

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type Records struct {
	mutex    sync.Mutex
	reqTotal uint32
	respInfo map[int16]uint32 // statuscode: total
}

func NewRecords() *Records {
	return &Records{
		respInfo: map[int16]uint32{},
	}
}

// records response info, notice not all request will send response
func (r *Records) RecordResponse(statuscode int16) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.respInfo[statuscode]; !ok {
		r.respInfo[statuscode] = 0
	}
	r.respInfo[statuscode] += 1
}

func (r *Records) RecordRequest() {
	atomic.AddUint32(&r.reqTotal, 1)
}

func (r *Records) ResponseInfo() (map[int16]uint32, uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	var total uint32 = 0
	m := make(map[int16]uint32, len(r.respInfo)) // do a copy
	for k, v := range r.respInfo {
		m[k] = v
		total += v
	}
	return m, total
}

func (r *Records) Requests() uint32 {
	return atomic.LoadUint32(&r.reqTotal)
}

func (r *Records) String() string {
	info, total := r.ResponseInfo()
	return fmt.Sprintf("request: %d, response total: %d, response status: %v", r.Requests(), total, info)
}

type ServerStats struct {
	connectionTotal  uint32
	connectionActive uint32
	connectionClosed uint32
	records          *Records
}

type ServerStatsReadOnly interface {
	ConnectionTotal() uint32
	ConnectionActive() uint32
	ConnectionClosed() uint32
	Requests() uint32
	ResponseInfo() (map[int16]uint32, uint32)
	String() string
}

func NewServerStats() *ServerStats {
	return &ServerStats{
		records: NewRecords(),
	}
}

func (s *ServerStats) ConnectionTotal() uint32 {
	return atomic.LoadUint32(&s.connectionTotal)
}

func (s *ServerStats) ConnectionActive() uint32 {
	return atomic.LoadUint32(&s.connectionActive)
}

func (s *ServerStats) ConnectionClosed() uint32 {
	return atomic.LoadUint32(&s.connectionClosed)
}

func (s *ServerStats) Records() *Records {
	return s.records
}

func (s *ServerStats) ResponseInfo() (map[int16]uint32, uint32) {
	return s.records.ResponseInfo()
}
func (s *ServerStats) Requests() uint32 {
	return s.records.Requests()
}

func (s *ServerStats) ActiveConnection() {
	atomic.AddUint32(&s.connectionTotal, 1)
	atomic.AddUint32(&s.connectionActive, 1)
}

func (s *ServerStats) CloseConnection() {
	// subtract x, add ^uint32(x-1)
	atomic.AddUint32(&s.connectionActive, ^uint32(0))
	atomic.AddUint32(&s.connectionClosed, 1)
}

func (s *ServerStats) String() string {
	return fmt.Sprintf("connections: { total: %d, actvie: %d, closed: %d}, response info: %s",
		s.ConnectionTotal(), s.ConnectionActive(), s.ConnectionClosed(), s.Records().String())
}

type ClientStatsReadOnly interface {
	ServerStatsReadOnly
	ExpectedResponseCount() uint32
	UnexpectedResponseCount() uint32
}

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

func (c *ClientStats) ExpectedResponseCount() uint32 {
	return atomic.LoadUint32(&c.expectedResponse)
}

func (c *ClientStats) UnexpectedResponseCount() uint32 {
	return atomic.LoadUint32(&c.unexpectedResponse)
}

func (c *ClientStats) Response(expected bool) {
	if expected {
		atomic.AddUint32(&c.expectedResponse, 1)
	} else {
		atomic.AddUint32(&c.unexpectedResponse, 1)
	}
}

func (c *ClientStats) String() string {
	return fmt.Sprintf("%s, expected response: %d, unexpected response: %d", c.ServerStats.String(), c.ExpectedResponseCount(), c.UnexpectedResponseCount())
}
