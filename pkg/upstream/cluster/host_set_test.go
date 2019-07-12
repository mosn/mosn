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

	v2 "sofastack.io/sofa-mosn/pkg/api/v2"
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

func TestHostSetDistinct(t *testing.T) {
	hs := &hostSet{}
	ip := "127.0.0.1"
	var hosts []types.Host
	for i := 0; i < 5; i++ {
		host := &mockHost{
			addr: ip,
		}
		hosts = append(hosts, host)
	}
	hs.setFinalHost(hosts)
	if len(hs.Hosts()) != 1 {
		t.Fatal("hostset distinct failed")
	}
}

// Test Refresh healthy hosts
func TestHostSetRefresh(t *testing.T) {
	hs := &hostSet{}
	configs := []simpleMockHostConfig{}
	for i := 10000; i < 10010; i++ {
		cfg := simpleMockHostConfig{
			addr:      fmt.Sprintf("127.0.0.1:%d", i),
			metaValue: "v1",
		}
		configs = append(configs, cfg)
	}
	for i := 11000; i < 11010; i++ {
		cfg := simpleMockHostConfig{
			addr:      fmt.Sprintf("127.0.0.1:%d", i),
			metaValue: "v2",
		}
		configs = append(configs, cfg)
	}
	var hosts []types.Host
	for _, cfg := range configs {
		h := newSimpleMockHost(cfg.addr, cfg.metaValue)
		hosts = append(hosts, h)
	}
	hs.setFinalHost(hosts)
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
	// mock health check set
	allHosts := hs.Hosts()
	host := allHosts[5] // mock one host healthy is changed
	host.SetHealthFlag(types.FAILED_ACTIVE_HC)
	hs.refreshHealthHost(host)
	if !(len(hs.HealthyHosts()) == 19 &&
		len(subV1.HealthyHosts()) == 9 &&
		len(subV2.HealthyHosts()) == 10) {
		t.Fatal("health check state changed not expected")
	}
}
