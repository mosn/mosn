package server

import (
	"crypto/tls"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

const testServerName = "test_server"

func setup() {
	handler := NewHandler(&mockClusterManagerFilter{}, &mockClusterManager{})
	initListenerAdapterInstance(testServerName, handler)
}

func TestMain(m *testing.M) {
	setup()
	m.Run()
}

func baseListenerConfig(addrStr string, name string) *v2.Listener {
	// add a new listener
	addr, _ := net.ResolveTCPAddr("tcp", addrStr)
	return &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:       name,
			BindToPort: true,
			AccessLogs: []v2.AccessLog{
				{
					Path:   "stdout",
					Format: types.DefaultAccessLogFormat,
				},
			},
			FilterChains: []v2.FilterChain{
				{
					FilterChainConfig: v2.FilterChainConfig{
						Filters: []v2.Filter{
							{
								Type: "network",
								Config: map[string]interface{}{
									"network": "exists",
								},
							}, // no network filter parsed, but the config still exists for test
						},
					},
					TLSContexts: []v2.TLSConfig{
						v2.TLSConfig{
							Status:     true,
							CACert:     mockCAPEM,
							CertChain:  mockCertPEM,
							PrivateKey: mockKeyPEM,
						},
					},
				},
			},
			StreamFilters: []v2.Filter{
				{
					Type: "stream",
					Config: map[string]interface{}{
						"stream": "exists",
					},
				},
			}, //no stream filters parsed, but the config still exists for test
		},
		Addr: addr,
		PerConnBufferLimitBytes: 1 << 15,
	}
}

// LDS include add\update\delete listener
func TestLDS(t *testing.T) {
	addrStr := "127.0.0.1:8080"
	name := "listener1"
	listenerConfig := baseListenerConfig(addrStr, name)
	// set a network filter do nothing, just for keep the connection not close
	nfcfs := []types.NetworkFilterChainFactory{
		&mockNetworkFilterFactory{},
	}
	if err := GetListenerAdapterInstance().AddOrUpdateListener(testServerName, listenerConfig, nfcfs, nil); err != nil {
		t.Fatalf("add a new listener failed", err)
	}
	time.Sleep(time.Second) // wait listener start
	// verify
	// add listener success
	handler := listenerAdapterInstance.defaultConnHandler.(*connHandler)
	if len(handler.listeners) != 1 {
		t.Fatalf("listener numbers is not expected %d", len(handler.listeners))
	}
	ln := handler.FindListenerByName(name)
	if ln == nil {
		t.Fatal("no listener found")
	}
	// use real connection to test
	// tls handshake success
	dialer := &net.Dialer{
		Timeout: time.Second,
	}
	if conn, err := tls.DialWithDialer(dialer, "tcp", addrStr, &tls.Config{
		InsecureSkipVerify: true,
	}); err != nil {
		t.Fatal("dial tls failed", err)
	} else {
		conn.Close()
	}
	// update listener
	// FIXME: update logger
	newListenerConfig := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name: name, // name should same as the exists listener
			AccessLogs: []v2.AccessLog{
				{},
			},
			FilterChains: []v2.FilterChain{
				{
					FilterChainConfig: v2.FilterChainConfig{
						Filters: []v2.Filter{}, // network filter will not be updated
					},
					TLSContexts: []v2.TLSConfig{ // only tls will be updated
						{
							Status: false,
						},
					},
				},
			},
			StreamFilters: []v2.Filter{}, // stream filter will not be updated
			Inspector:     true,
		},
		Addr: listenerConfig.Addr, // addr should not be changed
		PerConnBufferLimitBytes: 1 << 10,
	}
	if err := GetListenerAdapterInstance().AddOrUpdateListener(testServerName, newListenerConfig, nil, nil); err != nil {
		t.Fatal("update listener failed", err)
	}
	// verify
	// 1. listener have only 1
	if len(handler.listeners) != 1 {
		t.Fatalf("listener numbers is not expected %d", len(handler.listeners))
	}
	// 2. verify config, the updated configs should be changed, and the others should be same as old config
	newLn := handler.FindListenerByName(name)
	cfg := newLn.Config()
	if !(reflect.DeepEqual(cfg.FilterChains[0].TLSContexts[0], newListenerConfig.FilterChains[0].TLSContexts[0]) && //tls is new
		cfg.PerConnBufferLimitBytes == 1<<10 && // PerConnBufferLimitBytes is new
		cfg.Inspector && // inspector is new
		reflect.DeepEqual(cfg.FilterChains[0].Filters, listenerConfig.FilterChains[0].Filters) && // network filter is old
		reflect.DeepEqual(cfg.StreamFilters, listenerConfig.StreamFilters)) { // stream filter is old
		// FIXME: log config is new
		t.Fatal("new config is not expected")
	}
	// FIXME:
	// Logger level is new

	// 3. tls handshake should be failed, because tls is changed to false
	if conn, err := tls.DialWithDialer(dialer, "tcp", addrStr, &tls.Config{
		InsecureSkipVerify: true,
	}); err == nil {
		conn.Close()
		t.Fatal("listener should not be support tls any more")
	}
	// 4.common connection should be success, network filter will not be changed
	if conn, err := net.DialTimeout("tcp", addrStr, time.Second); err != nil {
		t.Fatal("dial listener failed", err)
	} else {
		conn.Close()
	}
	// test delete listener
	if err := GetListenerAdapterInstance().DeleteListener(testServerName, name); err != nil {
		t.Fatal("delete listener failed", err)
	}
	time.Sleep(time.Second) // wait listener close
	if len(handler.listeners) != 0 {
		t.Fatal("handler still have listener")
	}
	// dial should be failed
	if conn, err := net.DialTimeout("tcp", addrStr, time.Second); err == nil {
		conn.Close()
		t.Fatal("listener closed, dial should be failed")
	}
}

func TestUpdateTLS(t *testing.T) {
	addrStr := "127.0.0.1:8081"
	name := "listener2"
	listenerConfig := baseListenerConfig(addrStr, name)
	// set a network filter do nothing, just for keep the connection not close
	nfcfs := []types.NetworkFilterChainFactory{
		&mockNetworkFilterFactory{},
	}
	if err := GetListenerAdapterInstance().AddOrUpdateListener(testServerName, listenerConfig, nfcfs, nil); err != nil {
		t.Fatalf("add a new listener failed", err)
	}
	time.Sleep(time.Second) // wait listener start
	tlsCfg := v2.TLSConfig{
		Status: false,
	}
	// tls handleshake success
	dialer := &net.Dialer{
		Timeout: time.Second,
	}
	if conn, err := tls.DialWithDialer(dialer, "tcp", addrStr, &tls.Config{
		InsecureSkipVerify: true,
	}); err != nil {
		t.Fatal("dial tls failed", err)
	} else {
		conn.Close()
	}
	if err := GetListenerAdapterInstance().UpdateListenerTLS(testServerName, name, false, []v2.TLSConfig{tlsCfg}); err != nil {
		t.Fatalf("update tls listener failed", err)
	}
	handler := listenerAdapterInstance.defaultConnHandler.(*connHandler)
	newLn := handler.FindListenerByName(name)
	cfg := newLn.Config()
	// verify tls changed
	if !(reflect.DeepEqual(cfg.FilterChains[0].TLSContexts[0], tlsCfg) &&
		cfg.Inspector == false) {
		t.Fatal("update tls config not expected")
	}
	// tls handshake should be failed, because tls is changed to false
	if conn, err := tls.DialWithDialer(dialer, "tcp", addrStr, &tls.Config{
		InsecureSkipVerify: true,
	}); err == nil {
		conn.Close()
		t.Fatal("listener should not be support tls any more")
	}
}
