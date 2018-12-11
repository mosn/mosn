package healthcheck

import (
	"time"

	"github.com/alipay/sofa-mosn/pkg/types"
)

// use a mock session Factory to generate a mock session
type mockSessionFactory struct {
}

func (f *mockSessionFactory) NewSession(cfg map[string]interface{}, host types.Host) types.HealthCheckSession {
	return &mockSession{host}
}

type mockSession struct {
	host types.Host
}

func (s *mockSession) CheckHealth() bool {
	if mh, ok := s.host.(*mockHost); ok {
		if mh.delay > 0 {
			time.Sleep(mh.delay)
		}
	}
	return s.host.Health()
}
func (s *mockSession) OnTimeout() {}

type mockCluster struct {
	types.Cluster
	ps *mockPrioritySet
}

func (c *mockCluster) PrioritySet() types.PrioritySet {
	return c.ps
}

type mockPrioritySet struct {
	types.PrioritySet
	hs *mockHostSet
}

func (p *mockPrioritySet) HostSetsByPriority() []types.HostSet {
	return []types.HostSet{p.hs}
}

type mockHostSet struct {
	types.HostSet
	hosts []types.Host
}

func (hs *mockHostSet) Hosts() []types.Host {
	return hs.hosts
}

type mockHost struct {
	types.Host
	addr string
	flag uint64
	// mock status
	delay  time.Duration
	status bool
}

func (h *mockHost) Health() bool {
	return h.status
}

func (h *mockHost) AddressString() string {
	return h.addr
}

func (h *mockHost) ClearHealthFlag(flag types.HealthFlag) {
	h.flag &= ^uint64(flag)
}

func (h *mockHost) ContainHealthFlag(flag types.HealthFlag) bool {
	return h.flag&uint64(flag) > 0
}

func (h *mockHost) SetHealthFlag(flag types.HealthFlag) {
	h.flag |= uint64(flag)
}
