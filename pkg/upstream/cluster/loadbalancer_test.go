package cluster

import (
	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/types"
	"testing"
)

func TestNewLARBalancer(t *testing.T) {
	balancer := NewLoadBalancer(types.LeastActiveRequest, &hostSet{})
	assert.NotNil(t, balancer)
	assert.IsType(t, &leastActiveRequestLoadBalancer{}, balancer)
}

func TestLARChooseHost(t *testing.T) {
	hosts := createHostsetWithStats(exampleHostConfigs(), "test")
	balancer := NewLoadBalancer(types.LeastActiveRequest, hosts)
	host := balancer.ChooseHost(newMockLbContext(nil))
	assert.NotNil(t, host)

	expectHost := hosts.healthyHosts[0]
	for _, host := range hosts.healthyHosts[1:] {
		mockRequest(host, true, 10)
	}
	actual := balancer.ChooseHost(newMockLbContext(nil))
	assert.Equal(t, expectHost, actual)
}

func mockRequest(host types.Host, active bool, times int) {
	for i := 0; i < times; i++ {
		if active {
			host.HostStats().UpstreamRequestActive.Inc(1)
		}
	}
}
