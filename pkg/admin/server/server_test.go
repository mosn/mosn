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

package server

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	rawjson "encoding/json"

	"github.com/alipay/sofa-mosn/pkg/admin/store"
	"github.com/alipay/sofa-mosn/pkg/metrics"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
)

func getEffectiveConfig(port uint32) (string, error) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/config_dump", port))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("call admin api failed response status: %d, %s", resp.StatusCode, string(b)))
	}

	if err != nil {
		return "", err
	}
	return string(b), nil
}

func getStats(port uint32) (string, error) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/stats", port))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("call admin api failed response status: %d, %s", resp.StatusCode, string(b)))
	}

	if err != nil {
		return "", err
	}
	return string(b), nil
}

type mockMOSNConfig struct {
	Name string `json:"name"`
	Port uint32 `json:"port"`
}

func (m *mockMOSNConfig) GetAdmin() *v2.Admin {
	return &v2.Admin{
		Address: core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: 0,
					Address:  "",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: m.Port,
					},
					ResolverName: "",
					Ipv4Compat:   false,
				},
			},
		},
	}
}

func TestDumpConfig(t *testing.T) {
	server := Server{}
	config := &mockMOSNConfig{
		Name: "mock",
		Port: 8889,
	}
	server.Start(config)
	defer server.Close()

	if data, err := getEffectiveConfig(config.Port); err != nil {
		t.Error(err)
	} else {
		if data != `{"mosn_config":{"name":"mock","port":8889}}` {
			t.Errorf("unexpected effectiveConfig: %s\n", data)
		}
	}
	store.Reset()
}

func TestDumpStats(t *testing.T) {
	server := Server{}
	config := &mockMOSNConfig{
		Name: "mock",
		Port: 8889,
	}
	server.Start(config)
	defer server.Close()

	stats, _ := metrics.NewMetrics("DumpTest", map[string]string{"lbk1": "lbv1"})
	stats.Counter("ct1").Inc(1)
	stats.Gauge("gg2").Update(3)

	expected, _ := rawjson.MarshalIndent(map[string]map[string]map[string]string{
		"DumpTest": {
			"lbk1.lbv1": {
				"ct1": "1",
				"gg2": "3",
			},
		},
	}, "", "\t")

	if data, err := getStats(config.Port); err != nil {
		t.Error(err)
	} else {
		if data != string(expected) {
			t.Errorf("unexpected stats: %s\n", data)
		}
	}
	store.Reset()
}

// TestUpdateTLS see integrate test
