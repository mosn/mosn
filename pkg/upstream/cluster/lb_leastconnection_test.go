package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/types"
)

func TestLACChooseHost(t *testing.T) {
	hosts := createHostsetWithStats(exampleHostConfigs(), "test")
	balancer := NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveConnection}, hosts)
	host := balancer.ChooseHost(newMockLbContext(nil))
	assert.NotNil(t, host)

	hosts.Range(func(host types.Host) bool {
		mockRequest(host, true, 10)
		return true
	})
	// new lb to refresh edf
	balancer = NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveConnection}, hosts)
	actual := balancer.ChooseHost(newMockLbContext(nil))
	assert.NotNil(t, actual)

	// test only one host
	h := exampleHostConfigs()[0:1]
	hosts = createHostsetWithStats(h, "test")
	balancer = NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveConnection}, hosts)
	actual = balancer.ChooseHost(nil)
	assert.Equal(t, hosts.allHosts[0], actual)

	// test no host
	h = exampleHostConfigs()[0:0]
	hosts = createHostsetWithStats(h, "test")
	balancer = NewLoadBalancer(&clusterInfo{lbType: types.LeastActiveConnection}, hosts)
	actual = balancer.ChooseHost(nil)
	assert.Nil(t, actual)
}