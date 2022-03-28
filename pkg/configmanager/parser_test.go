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

package configmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

type testCallback struct {
	Count int
}

func (t *testCallback) ParsedCallback(data interface{}, endParsing bool) error {
	t.Count++
	return nil
}

var cb testCallback

func TestMain(m *testing.M) {
	RegisterConfigParsedListener(ParseCallbackKeyCluster, cb.ParsedCallback)
	RegisterConfigParsedListener(ParseCallbackKeyProcessor, cb.ParsedCallback)
	os.Exit(m.Run())
}

var mockedFilterChains = `
            {
             "match":"",
             "tls_context":{
               "status": true,
               "inspector": true,
               "server_name": "hello.com",
               "ca_cert": "-----BEGIN CERTIFICATE-----\nMIIDMjCCAhoCCQDaFC8PcSS5qTANBgkqhkiG9w0BAQsFADBbMQswCQYDVQQGEwJD\nTjEKMAgGA1UECAwBYTEKMAgGA1UEBwwBYTEKMAgGA1UECgwBYTEKMAgGA1UECwwB\nYTEKMAgGA1UEAwwBYTEQMA4GCSqGSIb3DQEJARYBYTAeFw0xODA2MTQwMjQyMjVa\nFw0xOTA2MTQwMjQyMjVaMFsxCzAJBgNVBAYTAkNOMQowCAYDVQQIDAFhMQowCAYD\nVQQHDAFhMQowCAYDVQQKDAFhMQowCAYDVQQLDAFhMQowCAYDVQQDDAFhMRAwDgYJ\nKoZIhvcNAQkBFgFhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArbNc\nmOXvvOqZgJdxMIiklE8lykVvz5d7QZ0+LDVG8phshq9/woigfB1aFBAI36/S5LZQ\n5Fd0znblSa+LOY06jdHTkbIBYFlxH4tdRaD0B7DbFzR5bpzLv2Q+Zf5u5RI73Nky\nH8CjW9QJjboArHkwm0YNeENaoR/96nYillgYLnunol4h0pxY7ZC6PpaB1EBaTXcz\n0iIUX4ktUJQmYZ/DFzB0oQl9IWOj18ml2wYzu9rYsySzj7EPnDOOebsRfS5hl3fz\nHi4TC4PDh0mQwHqDQ4ncztkybuRSXFQ6RzEPdR5qtp9NN/G/TlfyB0CET3AFmGkp\nE2irGoF/JoZXEDeXmQIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQApzhQLS7fAcExZ\nx1S+hcy7lLF8QcPlsiH32SnLFg5LPy4prz71mebUchmt97t4T3tSWzwXi8job7Q2\nONYc6sr1LvaFtg7qoCfz5fPP5x+kKDkEPwCDJSTVPcXP+UtA407pxX8KPRN8Roay\ne3oGcmNqVu/DkkufkIL3PBg41JEMovWtKD+PXmeBafc4vGCHSJHJBmzMe5QtwHA0\nss/A9LHPaq3aLcIyFr8x7clxc7zZVaim+lVfNV3oPBnB4gU7kLFVT0zOhkM+V1A4\nQ5GVbGAu4R7ItY8kJ2b7slre0ajPUp2FMregt4mnUM3mu1nbltVhtoknXqHHMGgN\n4Lh4JfNx\n-----END CERTIFICATE-----\n",
               "cert_chain": "-----BEGIN CERTIFICATE-----\nMIIDJTCCAg0CAQEwDQYJKoZIhvcNAQELBQAwWzELMAkGA1UEBhMCQ04xCjAIBgNV\nBAgMAWExCjAIBgNVBAcMAWExCjAIBgNVBAoMAWExCjAIBgNVBAsMAWExCjAIBgNV\nBAMMAWExEDAOBgkqhkiG9w0BCQEWAWEwHhcNMTgwNjE0MDMxMzQyWhcNMTkwNjE0\nMDMxMzQyWjBWMQswCQYDVQQGEwJDTjEKMAgGA1UECAwBYTEKMAgGA1UECgwBYTEK\nMAgGA1UECwwBYTERMA8GA1UEAwwIdGVzdC5jb20xEDAOBgkqhkiG9w0BCQEWAWEw\nggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCrPq+Mo0nS3dJU1qGFwlIB\ni9HqRm5RGcfps+0W5LjEhqUKxKUweRrwDaIxpiSqjKeehz9DtLUpXBD29pHuxODU\nVsMH2U1AGWn9l4jMnP6G5iTMPJ3ZTXszeqALe8lm/f807ZA0C7moc+t7/d3+b6d2\nlnwR+yWbIZJUu2qw+HrR0qPpNlBP3EMtlQBOqf4kCl6TfpqrGfc9lW0JjuE6Taq3\ngSIIgzCsoUFe30Yemho/Pp4zA9US97DZjScQLQAGiTsCRDBASxXGzODQOfZL3bCs\n2w//1lqGjmhp+tU1nR4MRN+euyNX42ioEz111iB8y0VzuTIsFBWwRTKK1SF7YSEb\nAgMBAAEwDQYJKoZIhvcNAQELBQADggEBABnRM9JJ21ZaujOTunONyVLHtmxUmrdr\n74OJW8xlXYEMFu57Wi40+4UoeEIUXHviBnONEfcITJITYUdqve2JjQsH2Qw3iBUr\nmsFrWS25t/Krk2FS2cKg8B9azW2+p1mBNm/FneMv2DMWHReGW0cBp3YncWD7OwQL\n9NcYfXfgBgHdhykctEQ97SgLHDKUCU8cPJv14eZ+ehIPiv8cDWw0mMdMeVK9q71Y\nWn2EgP7HzVgdbj17nP9JJjNvets39gD8bU0g2Lw3wuyb/j7CHPBBzqxh+a8pihI5\n3dRRchuVeMQkMuukyR+/A8UrBMA/gCTkXIcP6jKO1SkKq5ZwlMmapPc=\n-----END CERTIFICATE-----\n",
               "private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEAqz6vjKNJ0t3SVNahhcJSAYvR6kZuURnH6bPtFuS4xIalCsSl\nMHka8A2iMaYkqoynnoc/Q7S1KVwQ9vaR7sTg1FbDB9lNQBlp/ZeIzJz+huYkzDyd\n2U17M3qgC3vJZv3/NO2QNAu5qHPre/3d/m+ndpZ8EfslmyGSVLtqsPh60dKj6TZQ\nT9xDLZUATqn+JApek36aqxn3PZVtCY7hOk2qt4EiCIMwrKFBXt9GHpoaPz6eMwPV\nEvew2Y0nEC0ABok7AkQwQEsVxszg0Dn2S92wrNsP/9Zaho5oafrVNZ0eDETfnrsj\nV+NoqBM9ddYgfMtFc7kyLBQVsEUyitUhe2EhGwIDAQABAoIBAG2Bj5ca0Fmk+hzA\nh9fWdMSCWgE7es4n81wyb/nE15btF1t0dsIxn5VE0qR3P1lEyueoSz+LrpG9Syfy\nc03B3phKxzscrbbAybOeFJ/sASPYxk1IshRE5PT9hJzzUs6mvG1nQWDW4qmjP0Iy\nDKTpV6iRANQqy1iRtlay5r42l6vWwHfRfwAv4ExSS+RgkYcavqOp3e9If2JnFJuo\n7Zds2i7Ux8dURX7lHqKxTt6phgoMmMpvO3lFYVGos7F13OR9NKElzjiefAQbweAt\nt8R+6A1rlIwnfywxET9ZXglfOFK6Q0nqCJhcEcKzT/Xfkd+h9XPACjOObJh3a2+o\nwg9GBFECgYEA2a6JYuFanKzvajFPbSeN1csfI9jPpK2+tB5+BB72dE74B4rjygiG\n0Rb26UjovkYfJJqKuKr4zDL5ziSlJk199Ae2f6T7t7zmyhMlWQtVT12iTQvBINTz\nNerKi5HNVBsCSGj0snbwo8u4QRgTjaIoVqTlOlUQuGqYuZ75l8g35IkCgYEAyWOM\nKagzpGmHWq/0ThN4kkwWOdujxuqrPf4un2WXsir+L90UV7X9wY4mO19pe5Ga2Upu\nXFDsxAZsanf8SbzkTGHvzUobFL7eqsiwaUSGB/cGEtkIyVlAdyW9DhiZFt3i9mEF\nbBsHnEDHPHL4tu+BB8G3WahHjWOnbWZ3NTtP94MCgYEAi3XRmSLtjYER5cPvsevs\nZ7M5oRqvdT7G9divPW6k0MEjEJn/9BjgXqbKy4ylZ/m+zBGinEsVGKXz+wjpMY/m\nCOjEGCUYC5AfgAkiHVkwb6d6asgEFEe6BaoF18MyfBbNsJxlYMzowNeslS+an1vr\nYg9EuMl06+GHNSzPlVl1zZkCgYEAxXx8N2F9eu4NUK4ZafMIGpbIeOZdHbSERp+b\nAq5yasJkT3WB/F04QXVvImv3Gbj4W7r0rEyjUbtm16Vf3sOAMTMdIHhaRCbEXj+9\nVw1eTjM8XoE8b465e92jHk6a2WSvq6IK2i9LcDvJ5QptwZ7uLjgV37L4r7sYtVx0\n69uFGJcCgYEAot7im+Yi7nNsh1dJsDI48liKDjC6rbZoCeF7Tslp8Lt+4J9CA9Ux\nSHyKjg9ujbjkzCWrPU9hkugOidDOmu7tJAxB5cS00qJLRqB5lcPxjOWcBXCdpBkO\n0tdT/xRY/MYLf3wbT95enaPlhfeqBBXKNQDya6nISbfwbMLfNxdZPJ8=\n-----END RSA PRIVATE KEY-----\n",
               "verify_client": true,
               "cipher_suites": "ECDHE-RSA-AES256-GCM-SHA384",
               "ecdh_curves": "P256"
               },
               "filters": [
                {
                  "type": "proxy",
                  "config": {
                    "downstream_protocol": "SofaRpc",
                    "name": "proxy_config",
                    "upstream_protocol": "SofaRpc",
                    "router_config_name":"test_router"
                  }
                }
              ]
            }`

func TestParseClusterConfig(t *testing.T) {
	// Test Host Weight trans and hosts maps return
	cb.Count = 0
	cluster := v2.Cluster{
		Name: "test",
		Hosts: []v2.Host{
			{
				HostConfig: v2.HostConfig{
					Address: "127.0.0.1:80",
					Weight:  200,
				},
			},
			{
				HostConfig: v2.HostConfig{
					Address: "127.0.0.1:8080",
					Weight:  0,
				},
			},
			{
				HostConfig: v2.HostConfig{
					Address: "127.0.0.1:8080",
					Weight:  100,
				},
			},
		},
	}
	clusters, clMap := ParseClusterConfig([]v2.Cluster{cluster})
	c := clusters[0]
	for _, h := range c.Hosts {
		if h.Weight < v2.MinHostWeight || h.Weight > v2.MaxHostWeight {
			t.Error("unexpected host weight")
		}
	}
	if hosts, ok := clMap["test"]; !ok {
		t.Error("no cluster map")
	} else {
		if !reflect.DeepEqual(hosts, c.Hosts) {
			t.Error("unexpected hosts map")
		}
	}
	if cb.Count != 1 {
		t.Error("no callback")
	}

	if c.MaxRequestPerConn != v2.DefaultMaxRequestPerConn {
		t.Errorf("Expect cluster.MaxRequestPerConn default value %d but got %d",
			v2.DefaultMaxRequestPerConn, c.MaxRequestPerConn)
	}

	if c.ConnBufferLimitBytes != v2.DefaultConnBufferLimitBytes {
		t.Errorf("Expect cluster.ConnBufferLimitBytes default value%d, but got %d",
			v2.DefaultConnBufferLimitBytes, c.ConnBufferLimitBytes)
	}
}
func TestParseListenerConfig(t *testing.T) {
	// test listener inherit replace exists
	// make inherit listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Error(err)
		return
	}
	unixListener, err := net.Listen("unix", "/tmp/parse.sock")
	if err != nil {
		t.Error(err)
		return
	}
	defer unixListener.Close()
	defer listener.Close()
	tcpListener := listener.(*net.TCPListener)
	var inherit []net.Listener
	inherit = append(inherit, tcpListener, unixListener)

	lnStr := fmt.Sprintf(`{
		"address": "%s"
	}`, tcpListener.Addr().String())
	lc := &v2.Listener{}
	if err := json.Unmarshal([]byte(lnStr), lc); err != nil {
		t.Fatalf("listener config init failed: %v", err)
	}
	var inheritPacketConn []net.PacketConn
	ln := ParseListenerConfig(lc, inherit, inheritPacketConn)
	if !(ln.Addr != nil &&
		ln.Addr.String() == tcpListener.Addr().String() &&
		ln.PerConnBufferLimitBytes == 1<<15 &&
		ln.InheritListener != nil) {
		t.Errorf("listener parse unexpected, listener: %+v", ln)
	}

	// test unix
	lnStr = fmt.Sprintf(`{
		"address": "%s"
	}`, unixListener.Addr().String())
	unixlc := &v2.Listener{}
	unixlc.Network = "unix"
	if err := json.Unmarshal([]byte(lnStr), unixlc); err != nil {
		t.Fatalf("listener config init failed: %v", err)
	}

	ln = ParseListenerConfig(unixlc, inherit, inheritPacketConn)

	ll := unixListener.(*net.UnixListener)
	if !(ln.Addr != nil &&
		ln.Addr.String() == ll.Addr().String() &&
		ln.PerConnBufferLimitBytes == 1<<15 &&
		ln.InheritListener != nil) {
		t.Errorf("listener parse unexpected, listener: %+v", ln)
	}
}

func TestParseListenerUDP(t *testing.T) {
	packetconn, err := net.ListenPacket("udp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("listen packet error: %v", err)
	}
	ln := ParseListenerConfig(&v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			AddrConfig: "127.0.0.1:8080",
			Network:    "udp",
		},
	}, nil, []net.PacketConn{packetconn})
	if !(ln.Addr != nil &&
		ln.InheritPacketConn != nil) {
		t.Fatalf("parse udp listener failed: %+v", ln)
	}
}

func TestParseRouterConfig(t *testing.T) {
	filterStr := `{
		"filters": [{
			"type":"connection_manager",
			"config": {
				"router_config_name":"test_router",
				"virtual_hosts": []
			}
		}]
	}`
	filterChan := &v2.FilterChain{}
	if err := json.Unmarshal([]byte(filterStr), filterChan); err != nil {
		t.Fatal(err)
	}
	routerCfg, err := ParseRouterConfiguration(filterChan)
	if err != nil || routerCfg.RouterConfigName != "test_router" {
		t.Fatal("parse router configuration failed")
	}
	// test filter chain without router
	noRouteFilterChain := &v2.FilterChain{}
	if err := json.Unmarshal([]byte(mockedFilterChains), noRouteFilterChain); err != nil {
		t.Fatal(err)
	}
	emptyRouter, err := ParseRouterConfiguration(noRouteFilterChain)
	if err != nil || emptyRouter.RouterConfigName != "" {
		t.Fatal("parse no router configuration failed")
	}
}

func TestParseServerConfigWithAutoProc(t *testing.T) {
	// set env
	nc := runtime.NumCPU()
	// register cb
	cb := 0
	RegisterConfigParsedListener(ParseCallbackKeyProcessor, func(data interface{}, endParsing bool) error {
		p := data.(int)
		cb = p
		return nil
	})
	_ = ParseServerConfig(&v2.ServerConfig{
		Processor: "auto",
	})
	if cb != nc {
		t.Fatalf("processor callback should be called, cb:%d, numcpu:%d", cb, nc)
	}
}

func TestParseServerConfig(t *testing.T) {
	// set env
	os.Setenv("GOMAXPROCS", "1")
	// register cb
	cb := 0
	RegisterConfigParsedListener(ParseCallbackKeyProcessor, func(data interface{}, endParsing bool) error {
		p := data.(int)
		cb = p
		return nil
	})
	_ = ParseServerConfig(&v2.ServerConfig{
		Processor: 0,
	})
	if cb != 1 {
		t.Fatal("processor callback should be called")
	}
}

func TestListenerFilterFactories(t *testing.T) {
	api.RegisterListener("test1", func(cfg map[string]interface{}) (api.ListenerFilterChainFactory, error) {
		return &struct {
			api.ListenerFilterChainFactory
		}{}, nil
	})
	api.RegisterListener("test_nil", func(cfg map[string]interface{}) (api.ListenerFilterChainFactory, error) {
		return nil, nil
	})
	api.RegisterListener("test_error", func(cfg map[string]interface{}) (api.ListenerFilterChainFactory, error) {
		return nil, errors.New("invalid factory create")
	})

	config := []v2.Filter{
		{Type: "test1"},
		{Type: "test_error"},
		{Type: "not registered"},
		{Type: "test_nil"},
	}

	factoryNil := GetListenerFilterFactories("test_listener")
	assert.Nil(t, factoryNil)

	factory := AddOrUpdateListenerFilterFactories("test_listener", config)
	factory1 := GetListenerFilterFactories("test_listener")
	factory2 := GetListenerFilterFactories("test_listener")
	assert.NotNil(t, factory)
	assert.Equal(t, factory, factory1)
	assert.Equal(t, factory1, factory2)
}

func TestNetworkFilterFactories(t *testing.T) {
	api.RegisterNetwork("test_nil", func(cfg map[string]interface{}) (api.NetworkFilterChainFactory, error) {
		return nil, nil
	})
	api.RegisterNetwork("test1", func(cfg map[string]interface{}) (api.NetworkFilterChainFactory, error) {
		return &struct{ api.NetworkFilterChainFactory }{}, nil
	})
	api.RegisterNetwork("test_error", func(cfg map[string]interface{}) (api.NetworkFilterChainFactory, error) {
		return nil, errors.New("invalid factory create")
	})

	listenerConfig := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name: "test_listener",
			FilterChains: []v2.FilterChain{
				{
					FilterChainConfig: v2.FilterChainConfig{
						Filters: []v2.Filter{
							{Type: "test1"},
							{Type: "test_error"},
							{Type: "not registered"},
							{Type: "test_nil"},
						},
					},
				},
			},
		},
	}

	factoryNil := GetNetworkFilterFactories("test_listener")
	assert.Nil(t, factoryNil)

	factory := AddOrUpdateNetworkFilterFactories("test_listener", listenerConfig)
	factory1 := GetNetworkFilterFactories("test_listener")
	factory2 := GetNetworkFilterFactories("test_listener")
	assert.NotNil(t, factory)
	assert.Equal(t, factory, factory1)
	assert.Equal(t, factory1, factory2)
}
