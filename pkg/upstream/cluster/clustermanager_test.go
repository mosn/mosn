package cluster

import (
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func TestPrimaryCluster(t *testing.T) {
	pc := NewPrimaryCluster(&cluster{}, &v2.Cluster{}, true)
	if err := pc.UpdateCluster(&cluster{}, nil, true); err != nil {
		t.Error("update cluster failed")
	}
	hostsconfig := []v2.Host{host1, host2, host3, host4}
	var hosts []types.Host
	info := &clusterInfo{}
	for _, h := range hostsconfig {
		hosts = append(hosts, NewHost(h, info))
	}
	if err := pc.UpdateHosts(hosts); err != nil {
		t.Error("update hosts failed")
	}
}
