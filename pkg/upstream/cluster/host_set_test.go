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
	"fmt"
	"testing"

	"sofastack.io/sofa-mosn/pkg/api/v2"
	"sofastack.io/sofa-mosn/pkg/types"
)

func newSimpleMockHost(addr string, metaValue string) *mockHost {
	return &mockHost{
		addr: addr,
		meta: v2.Metadata{
			"key": metaValue,
		},
	}
}

type simpleMockHostConfig struct {
	addr      string
	metaValue string
}

func TestHostSet(t *testing.T) {
	hs := &hostSet{}
	configs := []simpleMockHostConfig{}
	for i := 10000; i < 10010; i++ {
		cfg := simpleMockHostConfig{
			addr:      fmt.Sprintf("%d", i),
			metaValue: "v1",
		}
		configs = append(configs, cfg)
	}
	for i := 11000; i < 11010; i++ {
		cfg := simpleMockHostConfig{
			addr:      fmt.Sprintf("%d", i),
			metaValue: "v2",
		}
		configs = append(configs, cfg)
	}
	var hosts []types.Host
	for _, cfg := range configs {
		h := newSimpleMockHost(cfg.addr, cfg.metaValue)
		hosts = append(hosts, h)
	}
	hs.UpdateHosts(hosts)
	// verify
	if !(len(hs.allHosts) == len(hs.healthyHosts) &&
		len(hs.allHosts) == 20) {
		t.Fatalf("update host set not expected, %v", hs)
	}
	// create subset
	subV1 := hs.createSubset(func(host types.Host) bool {
		if host.Metadata()["key"] == "v1" {
			return true
		}
		return false
	})
	subV2 := hs.createSubset(func(host types.Host) bool {
		if host.Metadata()["key"] == "v2" {
			return true
		}
		return false
	})
	// verify subv1 and subv2
	if !(len(subV1.Hosts()) == 10 &&
		len(subV2.Hosts()) == 10) {
		t.Fatalf("create sub host set failed")
	}
	for _, h := range subV1.Hosts() {
		if h.Metadata()["key"] != "v1" {
			t.Fatal("sub host set v1 got unepxected host")
		}
	}
	// update host, effect sub hostset
	var newHosts []types.Host
	newHosts = append(hosts[:5], hosts[10:]...)
	addHost := newSimpleMockHost("192.168.1.1", "v1")
	newHosts = append(newHosts, addHost)
	hs.UpdateHosts(newHosts)
	if !(len(hs.Hosts()) == len(hs.HealthyHosts()) &&
		len(hs.Hosts()) == 16 &&
		len(subV1.Hosts()) == len(subV1.HealthyHosts()) &&
		len(subV1.Hosts()) == 6 &&
		len(subV2.Hosts()) == len(subV2.HealthyHosts()) &&
		len(subV2.Hosts()) == 10) {
		t.Fatal("update hosts effected not expected")
	}
	// mock health check
	addHost.SetHealthFlag(types.FAILED_ACTIVE_HC)
	hs.refreshHealthHosts(addHost)
	if !(len(hs.HealthyHosts()) == 15 &&
		len(subV1.HealthyHosts()) == 5 &&
		len(subV2.HealthyHosts()) == 10) {
		t.Fatal("health check state changed not expected")
	}
}

func benchAddHost(b *testing.B, count int) {
	pool := makePool(2 * count)
	oldHosts := pool.MakeHosts(count, nil)
	newHosts := pool.MakeHosts(count, nil)
	newHosts = append(newHosts, oldHosts...)
	for i := 0; i < b.N; i++ {
		hs := &hostSet{
			allHosts: oldHosts,
		}
		hs.UpdateHosts(newHosts)
	}
}

// add and delete
func benchUpdateHost(b *testing.B, count int) {
	pool := makePool(3 * count)
	oldHosts := pool.MakeHosts(2*count, nil)
	newHosts := pool.MakeHosts(count, nil)
	newHosts = append(newHosts, oldHosts[:count]...)
	for i := 0; i < b.N; i++ {
		hs := &hostSet{
			allHosts: oldHosts,
		}
		hs.UpdateHosts(newHosts)

	}
}

func BenchmarkHostSetUpdateHost(b *testing.B) {

	b.Run("AddHost10", func(b *testing.B) {
		benchAddHost(b, 10)
	})

	b.Run("AddHost100", func(b *testing.B) {
		benchAddHost(b, 100)
	})

	b.Run("AddHost500", func(b *testing.B) {
		benchAddHost(b, 500)
	})

	b.Run("Update50", func(b *testing.B) {
		benchUpdateHost(b, 50)
	})

	b.Run("Update500", func(b *testing.B) {
		benchUpdateHost(b, 500)
	})

}

// Test Fast Remove
func TestHostSetRemoveHosts(t *testing.T) {
	// init hostset
	hs := &hostSet{}
	configs := []simpleMockHostConfig{}
	for i := 10000; i < 10010; i++ {
		cfg := simpleMockHostConfig{
			addr:      fmt.Sprintf("%d", i),
			metaValue: "v1",
		}
		configs = append(configs, cfg)
	}
	for i := 11000; i < 11010; i++ {
		cfg := simpleMockHostConfig{
			addr:      fmt.Sprintf("%d", i),
			metaValue: "v2",
		}
		configs = append(configs, cfg)
	}
	var hosts []types.Host
	for _, cfg := range configs {
		h := newSimpleMockHost(cfg.addr, cfg.metaValue)
		hosts = append(hosts, h)
	}
	hs.UpdateHosts(hosts)
	// create subset
	subV1 := hs.createSubset(func(host types.Host) bool {
		if host.Metadata()["key"] == "v1" {
			return true
		}
		return false
	})
	subV2 := hs.createSubset(func(host types.Host) bool {
		if host.Metadata()["key"] == "v2" {
			return true
		}
		return false
	})

	// remove host
	removed := []string{}
	for i := 10000; i < 10005; i++ {
		removed = append(removed, fmt.Sprintf("%d", i))
	}
	for i := 11000; i < 11009; i++ {
		removed = append(removed, fmt.Sprintf("%d", i))
	}
	hs.RemoveHosts(removed)
	// verify
	if !(len(hs.Hosts()) == len(hs.HealthyHosts()) &&
		len(hs.Hosts()) == 6 &&
		len(subV1.Hosts()) == len(subV1.HealthyHosts()) &&
		len(subV1.Hosts()) == 5 &&
		len(subV2.Hosts()) == len(subV2.HealthyHosts()) &&
		len(subV2.Hosts()) == 1) {
		t.Fatal("fast remove hosts not expected")
	}
}

func BenchmarkRemoveHosts(b *testing.B) {
	pool := makePool(100)
	totalHosts := pool.MakeHosts(100, nil)
	removedAddrs := []string{}
	for _, h := range totalHosts[:50] {
		removedAddrs = append(removedAddrs, h.AddressString())
	}
	b.Run("RemoveHosts", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hs := &hostSet{
				allHosts: totalHosts,
			}
			hs.RemoveHosts(removedAddrs)
		}
	})
}

func BenchmarkRefreshHost(b *testing.B) {
	pool := makePool(100)
	totalHosts := pool.MakeHosts(50, nil)
	totalHosts = append(totalHosts, pool.MakeHosts(50, v2.Metadata{
		"zone": "a",
	})...)
	hs := &hostSet{}
	hs.UpdateHosts(totalHosts)
	hs.createSubset(func(h types.Host) bool {
		if h.Metadata() != nil && h.Metadata()["zone"] == "a" {
			return true
		}
		return false
	})
	host := hs.Hosts()[55]
	b.Run("RefreshHost", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if i%2 == 0 {
				host.SetHealthFlag(types.FAILED_ACTIVE_HC)
			} else {
				host.ClearHealthFlag(types.FAILED_ACTIVE_HC)
			}
			hs.refreshHealthHosts(host)
		}
	})

}
