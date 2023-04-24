/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cluster

import (
	"reflect"
	"strings"
	"testing"
	"time"

	monkey "github.com/cch123/supermonkey"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/mosn/pkg/network"

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
	sdc := cluster.(*strictDnsCluster)

	monkey.PatchInstanceMethod(reflect.TypeOf(sdc.dnsResolver), "DnsResolve",
		func(resolver *network.DnsResolver, dnsAddr string, dnsLookupFamily v2.DnsLookupFamily) *[]network.DnsResponse {
			switch dnsAddr {
			case "github.com":
				return &[]network.DnsResponse{
					{Address: "127.0.0.1:10068"},
					{Address: "127.0.0.1:10069"},
				}
			case "www.baidu.com":
				return &[]network.DnsResponse{
					{Address: "127.0.0.1:10070"},
					{Address: "127.0.0.1:10071"},
				}
			case "www.qq.com:80":
				return &[]network.DnsResponse{
					{Address: "127.0.0.1:10072"},
					{Address: "127.0.0.1:10073"},
				}
			}
			return &[]network.DnsResponse{
				{Address: "127.0.0.1:10074"},
				{Address: "127.0.0.1:10075"},
			}
		})
	defer monkey.UnpatchAll()

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
	cluster.UpdateHosts(NewHostSet(hosts))
	// The domain will be re-parsed when UpdateHosts, So need sleep more time
	time.Sleep(3 * time.Second)
	// verify
	snap := cluster.Snapshot()
	hs := snap.HostSet()
	assert.Equal(t, hs.Size(), 6)
	snap.HostSet().Range(func(host types.Host) bool {
		if strings.Contains(host.AddressString(), host.Hostname()) {
			t.Errorf("[upstream][static_dns_cluster] Address %s not resolved.", host.AddressString())
		}
		return true
	})

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
	cluster.UpdateHosts(NewHostSet(hosts))
	// The domain will be re-parsed when UpdateHosts, So need sleep more time
	time.Sleep(3 * time.Second)
	cluster.Snapshot().HostSet().Range(func(host types.Host) bool {
		if strings.Contains(host.AddressString(), host.Hostname()) {
			t.Errorf("[upstream][static_dns_cluster] Address %s not resolved.", host.AddressString())
		}
		return true
	})

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
	cluster.UpdateHosts(NewHostSet(hosts))
	cluster.Snapshot().HostSet().Range(func(host types.Host) bool {
		if !strings.Contains(host.AddressString(), host.Hostname()) {
			t.Errorf("[upstream][static_dns_cluster] Address %s not resolved.", host.AddressString())
		}
		return true
	})

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
	if hostEqual(&hostSet{allHosts: host1}, NewHostSet(host2)) {
		t.Errorf("[upstream][strict dns cluster] hosts should not be equal.")
	}
	if hostEqual(&hostSet{allHosts: host2}, NewHostSet(host1)) {
		t.Errorf("[upstream][strict dns cluster] hosts should not be equal.")
	}
	if hostEqual(&hostSet{allHosts: host2}, NewHostSet(host3)) {
		t.Errorf("[upstream][strict dns cluster] hosts should not be equal.")
	}
	if !hostEqual(&hostSet{allHosts: host2}, NewHostSet(host2)) {
		t.Errorf("[upstream][strict dns cluster] hosts should be equal.")
	}
}

func TestMultipleDnsHostsInOneCluster(t *testing.T) {
	clusterConfig := v2.Cluster{
		Name:           "dynamic_cluster",
		ClusterType:    "STRICT_DNS",
		DnsRefreshRate: &api.DurationConfig{Duration: 5 * time.Second},
	}
	cluster := NewCluster(clusterConfig)
	sdc := cluster.(*strictDnsCluster)

	monkey.PatchInstanceMethod(reflect.TypeOf(sdc.dnsResolver), "DnsResolve",
		func(resolver *network.DnsResolver, dnsAddr string, dnsLookupFamily v2.DnsLookupFamily) *[]network.DnsResponse {
			if dnsAddr == "host1" {
				return &[]network.DnsResponse{
					{Address: "127.0.0.1:10068"},
					{Address: "127.0.0.1:10069"},
				}
			} else {
				return &[]network.DnsResponse{
					{Address: "127.0.0.1:10070"},
					{Address: "127.0.0.1:10071"},
				}
			}
		})
	defer monkey.UnpatchAll()

	addrs := []string{
		"host1:80",
		"host2:80",
	}
	var hosts []types.Host
	for _, addr := range addrs {
		host := v2.Host{
			HostConfig: v2.HostConfig{
				Address:  addr,
				Hostname: addr,
			},
		}
		h := NewSimpleHost(host, cluster.Snapshot().ClusterInfo())
		hosts = append(hosts, h)
	}

	cluster.UpdateHosts(NewHostSet(hosts))
	// The domain will be re-parsed when UpdateHosts, So need sleep more time
	time.Sleep(3 * time.Second)

	// verify
	snap := cluster.Snapshot()

	assert.Equal(t, snap.HostSet().Size(), 4)
}
