package wasm_test

import (
	_ "embed"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	config "mosn.io/mosn/pkg/config/v2"
	_ "mosn.io/mosn/pkg/filter/network/proxy"
	_ "mosn.io/mosn/pkg/stream/http"
	_ "mosn.io/mosn/pkg/stream/http2"
	"mosn.io/mosn/test/util/mosn"
)

//go:embed testdata/e2e/uri.wasm
var testBinE2EURI []byte

//go:embed testdata/e2e/protocol_version.wasm
var testBinE2EProtocolVersion []byte

//go:embed testdata/examples/router.wasm
var testBinExampleRouter []byte

//go:embed testdata/examples/wasi.wasm
var testBinExampleWASI []byte

var (
	requestBody  = "{\"hello\": \"panda\"}"
	responseBody = "{\"hello\": \"world\"}"

	serveJson = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Content-Type", "application/json")
		w.Write([]byte(responseBody)) // nolint
	})

	servePath = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Content-Type", "text/plain")
		w.Write([]byte(r.URL.String())) // nolint
	})
)

type testMosn struct {
	url     string
	logPath string
	*mosn.MosnWrapper
}

func TestURI(t *testing.T) {
	var backend *httptest.Server
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want, have := "/v1.0/hello?name=teddy", r.URL.RequestURI(); want != have {
			t.Fatalf("unexpected request URI, want: %q, have: %q", want, have)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if want, have := "/v1.0/hi?name=panda", string(body); want != have {
			t.Fatalf("unexpected request body, want: %q, have: %q", want, have)
		}
	})

	backend = httptest.NewServer(next)
	defer backend.Close()
	mosn := startMosn(t, backend.Listener.Addr().String(), testBinE2EURI)
	defer mosn.Close()

	resp, err := http.Get(mosn.url + "/v1.0/hi?name=panda")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
}

func TestProtocolVersion(t *testing.T) {
	tests := []struct {
		http2    bool
		respBody string
	}{
		{
			http2:    false,
			respBody: "HTTP/1.1",
		},
		// TODO: Enable http/2
	}

	backend := httptest.NewServer(serveJson)
	defer backend.Close()
	mosn := startMosn(t, backend.Listener.Addr().String(), testBinE2EProtocolVersion)
	defer mosn.Close()

	for _, tc := range tests {
		tt := tc
		t.Run(tt.respBody, func(t *testing.T) {
			resp, err := http.Get(mosn.url)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if want, have := tt.respBody, string(body); want != have {
				t.Errorf("unexpected response body, want: %q, have: %q", want, have)
			}
		})
	}
}

func TestExampleWASI(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Set-Cookie", "a=b") // example of multiple headers
		w.Header().Add("Set-Cookie", "c=d")
		w.Header().Set("Date", "Tue, 15 Nov 1994 08:12:31 GMT")
		w.Write([]byte(`{"hello": "world"}`)) // nolint
	}))
	defer backend.Close()

	stdout, stderr := CaptureStdio(t, func() {
		mosn := startMosn(t, backend.Listener.Addr().String(), testBinExampleWASI)
		defer mosn.Close()

		// Make a client request which should print to the console
		req, err := http.NewRequest("POST", mosn.url, strings.NewReader(requestBody))
		if err != nil {
			log.Panicln(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Host = "localhost"
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Panicln(err)
		}
		defer resp.Body.Close()
	})

	require.Equal(t, stdout, `POST / HTTP/1.1
host: localhost
content-length: 18
content-type: application/json
user-agent: Go-http-client/1.1
accept-encoding: gzip

{"hello": "panda"}

HTTP/1.1 200
content-length: 18
content-type: application/json
set-cookie: a=b
set-cookie: c=d
date: Tue, 15 Nov 1994 08:12:31 GMT

{"hello": "world"}
`)
	require.Zero(t, stderr)
}

func TestExampleRouter(t *testing.T) {
	tests := []struct {
		path     string
		respBody string
	}{
		{
			path:     "/",
			respBody: "hello world",
		},
		{
			path:     "/nothosst",
			respBody: "hello world",
		},
		{
			path:     "/host/a",
			respBody: "/a",
		},
		{
			path:     "/host/b?name=panda",
			respBody: "/b?name=panda",
		},
	}

	backend := httptest.NewServer(servePath)
	defer backend.Close()
	mosn := startMosn(t, backend.Listener.Addr().String(), testBinExampleRouter)
	defer mosn.Close()

	for _, tc := range tests {
		tt := tc
		t.Run(tt.path, func(t *testing.T) {
			url := fmt.Sprintf("%s%s", mosn.url, tt.path)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			content, _ := io.ReadAll(resp.Body)
			if have, want := string(content), tt.respBody; have != want {
				t.Errorf("have %s, want %s", have, want)
			}
		})
	}
}

func startMosn(t *testing.T, backendAddr string, wasm []byte) testMosn {
	t.Helper()

	port := freePort()
	adminPort := freePort()

	logPath := filepath.Join(t.TempDir(), "httpwasm.log")
	wasmPath := filepath.Join(t.TempDir(), "test.wasm")
	if err := os.WriteFile(wasmPath, wasm, 0o600); err != nil {
		t.Fatal(err)
	}

	app := mosn.NewMosn(&config.MOSNConfig{
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
							StreamFilters: []config.Filter{
								{
									Type: "httpwasm",
									Config: map[string]interface{}{
										"path":   wasmPath,
										"config": "open sesame",
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
	})
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
			}
		}
	}
	t.Fatal("httpwasm start failed")
	return testMosn{}
}

func freePort() int {
	l, _ := net.Listen("tcp", ":0")
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
