package cluster

import (
	"reflect"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func TestPrioritySet_GetHostInfo(t *testing.T) {
	ps := prioritySet{}
	hs := ps.GetOrCreateHostSet(0)
	hostscfg := []v2.Host{
		{
			HostConfig: v2.HostConfig{
				Address: "127.0.0.1",
			},
			MetaData: v2.Metadata{
				"zone": "test1",
			},
		},
		{
			HostConfig: v2.HostConfig{
				Address: "127.0.0.1",
			},
			MetaData: v2.Metadata{
				"zone": "test2",
			},
		},
	}
	info := &clusterInfo{
		name: "test",
	}
	var hosts []types.Host
	for _, cfg := range hostscfg {
		hosts = append(hosts, NewHost(cfg, info))
	}
	hs.UpdateHosts(hosts, nil, nil, nil, nil, nil)
	hostinfo := ps.GetHostsInfo(0)
	if len(hostinfo) != len(hostscfg) {
		t.Error("hostinfo length not expected")
	}
	for i := range hostinfo {
		cfgMeta := hostscfg[i].MetaData
		getMeta := hostinfo[i].OriginMetaData()
		if !reflect.DeepEqual(cfgMeta, getMeta) {
			t.Errorf("hostconfig is %v, but got %v", cfgMeta, getMeta)
		}
	}
}
