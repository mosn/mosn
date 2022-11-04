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

package wasm_test

import (
	_ "embed"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	config "mosn.io/mosn/pkg/config/v2"
	_ "mosn.io/mosn/pkg/filter/network/proxy"
	_ "mosn.io/mosn/pkg/filter/stream/httpwasm"
	_ "mosn.io/mosn/pkg/stream/http"
	_ "mosn.io/mosn/pkg/stream/http2"
	"mosn.io/mosn/test/util/mosn"
	"mosn.io/pkg/log"
)

func init() {
	log.DefaultLogger.SetLogLevel(log.ERROR)
}

type testMosn struct {
	url     string
	logPath string
	*mosn.MosnWrapper
}

// can't use go:embed as it is in a different module
var pathRequestHeader = "testdata/req-header/main.wasm"

func Test_HttpWasm(t *testing.T) {
	var backend *httptest.Server
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Wasm-Context") == "" {
			t.Fatalf("expected to see request header from wasm: %v", r.Header)
		}
	})

	backend = httptest.NewServer(next)
	defer backend.Close()

	mosn, err := startMosn(backend.Listener.Addr().String(), pathRequestHeader, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer mosn.Close()

	resp, err := http.Get(mosn.url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
}

func Benchmark_Baseline(b *testing.B) {
	benchmark(b, "")
}

func Benchmark_HttpWasm(b *testing.B) {
	benchmark(b, pathRequestHeader)
}

func benchmark(b *testing.B, wasmPath string) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer backend.Close()

	mosn, err := startMosn(backend.Listener.Addr().String(), wasmPath, b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	defer mosn.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Get(mosn.url)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func startMosn(backendAddr, wasmPath, tmpDir string) (testMosn, error) {
	port := freePort()
	adminPort := freePort()
	logPath := filepath.Join(tmpDir, "httpwasm.log")

	c := &config.MOSNConfig{
		Pid: filepath.Join(tmpDir, "mosn.pid"),
		Servers: []config.ServerConfig{
			{
				DefaultLogPath:  logPath,
				DefaultLogLevel: "INFO",
				Routers: []*config.RouterConfiguration{
					{
						RouterConfigurationConfig: config.RouterConfigurationConfig{
							RouterConfigName: "server_router",
						},
						VirtualHosts: []config.VirtualHost{
							{
								Name:    "serverHost",
								Domains: []string{"*"},
								Routers: []config.Router{
									{
										RouterConfig: config.RouterConfig{
											Match: config.RouterMatch{
												Prefix: "/",
											},
											Route: config.RouteAction{
												RouterActionConfig: config.RouterActionConfig{
													ClusterName: "serverCluster",
												},
											},
										},
									},
								},
							},
						},
					},
				},
				Listeners: []config.Listener{
					{
						ListenerConfig: config.ListenerConfig{
							Name:       "serverListener",
							AddrConfig: fmt.Sprintf("127.0.0.1:%d", port),
							BindToPort: true,
							FilterChains: []config.FilterChain{
								{
									FilterChainConfig: config.FilterChainConfig{
										Filters: []config.Filter{
											{
												Type: "proxy",
												Config: map[string]interface{}{
													"downstream_protocol": "Http1",
													"upstream_protocol":   "Http1",
													"router_config_name":  "server_router",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		ClusterManager: config.ClusterManagerConfig{
			Clusters: []config.Cluster{
				{
					Name:                 "serverCluster",
					ClusterType:          "SIMPLE",
					LbType:               "LB_RANDOM",
					MaxRequestPerConn:    1024,
					ConnBufferLimitBytes: 32768,
					Hosts: []config.Host{
						{
							HostConfig: config.HostConfig{
								Address: backendAddr,
							},
						},
					},
				},
			},
		},
		RawAdmin: &config.Admin{
			Address: &config.AddressInfo{
				SocketAddress: config.SocketAddress{
					Address:   "127.0.0.1",
					PortValue: uint32(adminPort),
				},
			},
		},
		DisableUpgrade: true,
	}

	if wasmPath != "" {
		c.Servers[0].Listeners[0].ListenerConfig.StreamFilters = []config.Filter{
			{
				Type: "httpwasm",
				Config: map[string]interface{}{
					"path":   wasmPath,
					"config": "open sesame",
				},
			},
		}
	}

	app := mosn.NewMosn(c)
	app.Start()
	for i := 0; i < 100; i++ {
		time.Sleep(200 * time.Millisecond)
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d", adminPort))
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			time.Sleep(1 * time.Second)
			return testMosn{
				url:         fmt.Sprintf("http://127.0.0.1:%d", port),
				logPath:     logPath,
				MosnWrapper: app,
			}, nil
		}
	}
	return testMosn{}, errors.New("httpwasm start failed")
}

func freePort() int {
	l, _ := net.Listen("tcp", ":0")
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
