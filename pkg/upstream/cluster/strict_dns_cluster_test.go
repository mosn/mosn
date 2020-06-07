package cluster

import (
	"mosn.io/api"
	"strings"
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

func _createDynamicCluster() types.Cluster {
	clusterConfig := v2.Cluster{
		Name:           "dynamic_cluster",
		LbType:         v2.LB_ROUNDROBIN,
		ClusterType:    "STRICT_DNS",
		RespectDnsTTL:  true,
		DnsRefreshRate: &api.DurationConfig{5 * time.Second},
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy: 1, // AnyEndPoint
			SubsetSelectors: [][]string{
				[]string{"version"},
				[]string{"version", "zone"},
			},
		},
	}
	return NewCluster(clusterConfig)
}

func TestDynamicClusterUpdateHosts(t *testing.T) {
	cluster := _createDynamicCluster()
	// init hosts
	addrs := []string{
		"github.com:80",
		"www.baidu.com:80",
		"www.qq.com:80",
	}
	var hosts []types.Host
	metas := []api.Metadata{
		api.Metadata{"version": "1", "zone": "a"},
		api.Metadata{"version": "1", "zone": "b"},
		api.Metadata{"version": "2", "zone": "a"},
	}
	for i, meta := range metas {
		host := v2.Host{
			HostConfig: v2.HostConfig{
				Address:        addrs[i],
				Hostname:       addrs[i],
				Weight:         0,
				MetaDataConfig: nil,
				TLSDisable:     false,
			},
			MetaData: meta,
		}
		h := NewSimpleHost(host, cluster.Snapshot().ClusterInfo())
		hosts = append(hosts, h)
	}
	cluster.UpdateHosts(hosts)
	// The domain will be re-parsed when UpdateHosts, So need sleep more time
	time.Sleep(3 * time.Second)
	// verify
	snap := cluster.Snapshot()
	hostSet := snap.HostSet().Hosts()
	for _, host := range hostSet {
		if strings.Contains(host.AddressString(), host.Hostname()) {
			t.Errorf("[upstream][static_dns_cluster] Address %s not resolved.", host.AddressString())
		}
	}

	host := v2.Host{
		HostConfig: v2.HostConfig{
			Address:        "www.test.com:80",
			Hostname:       "www.test.com",
			Weight:         0,
			MetaDataConfig: nil,
			TLSDisable:     false,
		},
	}
	h := NewSimpleHost(host, cluster.Snapshot().ClusterInfo())
	hosts = []types.Host{h}
	cluster.UpdateHosts(hosts)
	// The domain will be re-parsed when UpdateHosts, So need sleep more time
	time.Sleep(3 * time.Second)
	hostSet = cluster.Snapshot().HostSet().Hosts()
	for _, host := range hostSet {
		if strings.Contains(host.AddressString(), host.Hostname()) {
			t.Errorf("[upstream][static_dns_cluster] Address %s not resolved.", host.AddressString())
		}
	}

	host = v2.Host{
		HostConfig: v2.HostConfig{
			Address:        "127.0.0.1:80",
			Hostname:       "127.0.0.1",
			Weight:         0,
			MetaDataConfig: nil,
			TLSDisable:     false,
		},
	}

	h = NewSimpleHost(host, cluster.Snapshot().ClusterInfo())
	hosts = []types.Host{h}
	cluster.UpdateHosts(hosts)
	hostSet = cluster.Snapshot().HostSet().Hosts()
	for _, host := range hostSet {
		if !strings.Contains(host.AddressString(), host.Hostname()) {
			t.Errorf("[upstream][static_dns_cluster] Address %s not resolved.", host.AddressString())
		}
	}

	ip1 := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}
	ip2 := []string{"1.1.1.1", "2.2.2.2"}
	ip3 := []string{"1.1.1.1", "2.2.2.2"}
	var host1 []types.Host
	var host2 []types.Host
	var host3 []types.Host
	for _, ip := range ip1 {
		host := &simpleHost{
			addressString: ip,
		}
		host1 = append(host1, host)
	}
	for _, ip := range ip2 {
		host := &simpleHost{
			addressString: ip,
		}
		host2 = append(host2, host)
	}
	for i, ip := range ip3 {
		host := &simpleHost{
			addressString: ip,
			metaData:      metas[i],
		}
		host3 = append(host3, host)
	}
	if hostEqual(&host1, &host2) {
		t.Errorf("[upstream][strict dns cluster] hosts should not be equal.")
	}
	if hostEqual(&host2, &host1) {
		t.Errorf("[upstream][strict dns cluster] hosts should not be equal.")
	}
	if hostEqual(&host2, &host3) {
		t.Errorf("[upstream][strict dns cluster] hosts should not be equal.")
	}
	if !hostEqual(&host2, &host2) {
		t.Errorf("[upstream][strict dns cluster] hosts should be equal.")
	}
}
