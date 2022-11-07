//go:build wasmer
// +build wasmer

package wasm_test

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tetratelabs/wabin/binary"
	"github.com/tetratelabs/wabin/wasm"

	config "mosn.io/mosn/pkg/config/v2"
	_ "mosn.io/mosn/pkg/filter/network/proxy"
	_ "mosn.io/mosn/pkg/filter/stream/proxywasm"
	_ "mosn.io/mosn/pkg/stream/http"
	_ "mosn.io/mosn/pkg/stream/http2"
	_ "mosn.io/mosn/pkg/wasm/runtime/wasmer"
	"mosn.io/mosn/test/util/mosn"
)

type testMosn struct {
	url     string
	logPath string
	*mosn.MosnWrapper
}

const pathResponseHeaderV1 = "testdata/req-header-v1/main.wasm"

func Test_ProxyWasmV1(t *testing.T) {
	// Ensure the module was compiled with the correct ABI as this is hard to verify at runtime.
	requireModuleExport(t, pathResponseHeaderV1, "proxy_abi_version_0_1_0")

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Wasm-Context") == "" {
			t.Fatalf("expected to see request header from wasm: %v", r.Header)
		}
	}))
	defer backend.Close()
	logPath := filepath.Join(t.TempDir(), "mosn.log")

	mosn, err := startMosn(backend.Listener.Addr().String(), pathResponseHeaderV1, logPath)
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

func Benchmark_BaseCase(b *testing.B) {
	benchmark(b, "")
}

func Benchmark_ProxyWasmV1(b *testing.B) {
	benchmark(b, pathResponseHeaderV1)
}

func benchmark(b *testing.B, wasmPath string) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer backend.Close()
	logPath := filepath.Join(b.TempDir(), "mosn.log")

	mosn, err := startMosn(backend.Listener.Addr().String(), wasmPath, logPath)
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

func startMosn(backendAddr string, wasmPath, logPath string) (testMosn, error) {
	port := freePort()
	adminPort := freePort()
	c := &config.MOSNConfig{
		Servers: []config.ServerConfig{
			{
				DefaultLogPath:  logPath,
				DefaultLogLevel: "ERROR",
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
				Type: "proxywasm",
				Config: map[string]interface{}{
					"instance_num": 1,
					"vm_config": map[string]interface{}{
						"engine": "wasmer",
						"path":   wasmPath,
					},
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
	return testMosn{}, errors.New("mosn start failed")
}

func freePort() int {
	l, _ := net.Listen("tcp", ":0")
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func requireModuleExport(t *testing.T, wasmPath, want string) {
	bin, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := binary.DecodeModule(bin, wasm.CoreFeaturesV2)
	var exports []string
	for _, e := range mod.ExportSection {
		if e.Name == want {
			return
		}
		exports = append(exports, e.Name)
	}
	t.Errorf("export not found, want: %v, have: %v", want, exports)
}
