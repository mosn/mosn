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

package mtls

import (
	"io/ioutil"
	"testing"
	"time"

	"mosn.io/mosn/pkg/config/v2"
)

// Test the tls functions in static mode.

// TestServerContextManagerWithMultipleCert tests the contextManager's core logic
// make three certificates with different dns and common name
// test context manager can find correct certificate for different client
func TestServerContextManagerWithMultipleCert(t *testing.T) {
	var filterChains []v2.FilterChain
	testCases := []struct {
		Info *certInfo
		Addr string
	}{
		{Info: &certInfo{"Cert1", "RSA", "www.example.com"}, Addr: "www.example.com"},
		{Info: &certInfo{"Cert2", "RSA", "*.example.com"}, Addr: "test.example.com"},
		{Info: &certInfo{"Cert3", "P256", "*.com"}, Addr: "www.foo.com"},
	}
	for i, tc := range testCases {
		cfg, err := tc.Info.CreateCertConfig()
		if err != nil {
			t.Errorf("#%d %v", i, err)
			return
		}
		fc := v2.FilterChain{
			TLSContexts: []v2.TLSConfig{
				*cfg,
			},
		}
		filterChains = append(filterChains, fc)
	}
	lc := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			FilterChains: filterChains,
		},
	}
	ctxMng, err := NewTLSServerContextManager(lc)
	if err != nil {
		t.Errorf("create context manager failed %v", err)
		return
	}
	server := MockServer{
		Mng: ctxMng,
		t:   t,
	}
	server.GoListenAndServe(t)
	defer server.Close()
	time.Sleep(time.Second) //wait server start
	// request with different "servername"
	// context manager just find a certificate to response
	// the certificate may be not match the client
	for i, tc := range testCases {
		cfg := &v2.TLSConfig{
			Status:       true,
			ServerName:   tc.Addr,
			InsecureSkip: true,
		}
		cltMng, err := NewTLSClientContextManager(cfg)
		if err != nil {
			t.Errorf("create client context manager failed %v", err)
			continue
		}
		resp, err := MockClient(t, server.Addr, cltMng)
		if err != nil {
			t.Errorf("#%d request server error %v", i, err)
			continue
		}

		serverCN := resp.TLS.PeerCertificates[0].Subject.CommonName
		if serverCN != tc.Info.CommonName {
			t.Errorf("#%d expected request server config %s , but got %s", i, tc.Info.CommonName, serverCN)
		}

		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}
	// request a unknown server name, return the first certificate
	// static provider is always ready
	cfg := &v2.TLSConfig{
		Status:       true,
		ServerName:   "www.example.net",
		InsecureSkip: true,
	}
	cltMng, err := NewTLSClientContextManager(cfg)
	if err != nil {
		t.Errorf("create client context manager failed %v", err)
		return
	}
	resp, err := MockClient(t, server.Addr, cltMng)
	if err != nil {
		t.Errorf("request server error %v", err)
		return
	}
	defer resp.Body.Close()

	serverCN := resp.TLS.PeerCertificates[0].Subject.CommonName
	expected := testCases[0].Info.CommonName
	if serverCN != expected {
		t.Errorf("expected request server config  %s , but got %s", expected, serverCN)
	}
	ioutil.ReadAll(resp.Body)

}

// TestVerifyClient tests a client must have certificate to server
func TestVerifyClient(t *testing.T) {
	info := &certInfo{
		CommonName: "test",
		Curve:      "P256",
	}
	cfg, err := info.CreateCertConfig()
	if err != nil {
		t.Error(err)
		return
	}
	cfg.RequireClientCert = true
	cfg.VerifyClient = true
	filterChains := []v2.FilterChain{
		{
			TLSContexts: []v2.TLSConfig{
				*cfg,
			},
		},
	}
	lc := &v2.Listener{}
	lc.FilterChains = filterChains
	ctxMng, err := NewTLSServerContextManager(lc)
	if err != nil {
		t.Errorf("create context manager failed %v", err)
		return
	}
	server := MockServer{
		Mng: ctxMng,
		t:   t,
	}
	server.GoListenAndServe(t)
	defer server.Close()
	time.Sleep(time.Second) //wait server start
	clientConfigs := []*v2.TLSConfig{
		// Verify Server
		{
			Status:     true,
			CACert:     cfg.CACert,
			CertChain:  cfg.CertChain,
			PrivateKey: cfg.PrivateKey,
			ServerName: "127.0.0.1",
		},
		// Skip Verify Server
		{
			Status:       true,
			CertChain:    cfg.CertChain,
			PrivateKey:   cfg.PrivateKey,
			InsecureSkip: true,
		},
	}
	for i, cfg := range clientConfigs {
		cltMng, err := NewTLSClientContextManager(cfg)
		if err != nil {
			t.Errorf("#%d create client context manager failed %v", i, err)
			continue
		}

		resp, err := MockClient(t, server.Addr, cltMng)
		if err != nil {
			t.Errorf("request server error %v", err)
			continue
		}
		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}

	cfg = &v2.TLSConfig{
		Status:       true,
		ServerName:   "127.0.0.1",
		InsecureSkip: true,
	}
	cltMng, err := NewTLSClientContextManager(cfg)
	if err != nil {
		t.Errorf("create client context manager failed %v", err)
		return
	}

	resp, err := MockClient(t, server.Addr, cltMng)
	// expected bad certificate
	if err == nil {
		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		t.Errorf("server should verify client certificate")
		return
	}
}

// TestInspector tests context manager support both tls and non-tls
func TestInspector(t *testing.T) {
	info := &certInfo{
		CommonName: "test",
		Curve:      "P256",
		DNS:        "test",
	}
	cfg, err := info.CreateCertConfig()
	if err != nil {
		t.Error(err)
		return
	}
	cfg.VerifyClient = true
	filterChains := []v2.FilterChain{
		{
			TLSContexts: []v2.TLSConfig{
				*cfg,
			},
		},
	}
	lc := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Inspector:    true,
			FilterChains: filterChains,
		},
	}
	ctxMng, err := NewTLSServerContextManager(lc)
	if err != nil {
		t.Errorf("create context manager failed %v", err)
		return
	}
	server := MockServer{
		Mng: ctxMng,
		t:   t,
	}
	server.GoListenAndServe(t)
	defer server.Close()
	time.Sleep(time.Second) //wait server start
	cltMng, err := NewTLSClientContextManager(&v2.TLSConfig{
		Status:     true,
		CACert:     cfg.CACert,
		CertChain:  cfg.CertChain,
		PrivateKey: cfg.PrivateKey,
		ServerName: "test",
	})
	if err != nil {
		t.Errorf("create client context manager failed %v", err)
		return
	}
	// non-tls
	resp, err := MockClient(t, server.Addr, nil)
	if err != nil {
		t.Errorf("request server error %v", err)
		return
	}
	ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	// tls
	resp, err = MockClient(t, server.Addr, cltMng)
	if err != nil {
		t.Errorf("request server error %v", err)
		return
	}
	ioutil.ReadAll(resp.Body)
	resp.Body.Close()

}

// Test one filter chain contains multiple certificates
func TestServerContextManagerWithMultipleCertInOneFilterChain(t *testing.T) {
	testCases := []struct {
		Info *certInfo
		Addr string
	}{
		{Info: &certInfo{"Cert1", "RSA", "www.example.com"}, Addr: "www.example.com"},
		{Info: &certInfo{"Cert2", "RSA", "*.example.com"}, Addr: "test.example.com"},
		{Info: &certInfo{"Cert3", "P256", "*.com"}, Addr: "www.foo.com"},
	}
	tlsContexts := []v2.TLSConfig{}
	for i, tc := range testCases {
		cfg, err := tc.Info.CreateCertConfig()
		if err != nil {
			t.Errorf("#%d %v", i, err)
			return
		}
		tlsContexts = append(tlsContexts, *cfg)
	}
	filterChains := []v2.FilterChain{
		{
			TLSContexts: tlsContexts,
		},
	}
	lc := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			FilterChains: filterChains,
		},
	}
	ctxMng, err := NewTLSServerContextManager(lc)
	if err != nil {
		t.Errorf("create context manager failed %v", err)
		return
	}
	server := MockServer{
		Mng: ctxMng,
		t:   t,
	}
	server.GoListenAndServe(t)
	defer server.Close()
	time.Sleep(time.Second) //wait server start
	// request with different "servername"
	// context manager just find a certificate to response
	// the certificate may be not match the client
	for i, tc := range testCases {
		cfg := &v2.TLSConfig{
			Status:       true,
			ServerName:   tc.Addr,
			InsecureSkip: true,
		}
		cltMng, err := NewTLSClientContextManager(cfg)
		if err != nil {
			t.Errorf("create client context manager failed %v", err)
			continue
		}
		resp, err := MockClient(t, server.Addr, cltMng)
		if err != nil {
			t.Errorf("#%d request server error %v", i, err)
			continue
		}

		serverCN := resp.TLS.PeerCertificates[0].Subject.CommonName
		if serverCN != tc.Info.CommonName {
			t.Errorf("#%d expected request server config %s , but got %s", i, tc.Info.CommonName, serverCN)
		}

		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}
	// request a unknown server name, return the first certificate
	cfg := &v2.TLSConfig{
		Status:       true,
		ServerName:   "www.example.net",
		InsecureSkip: true,
	}
	cltMng, err := NewTLSClientContextManager(cfg)
	if err != nil {
		t.Errorf("create client context manager failed %v", err)
		return
	}
	resp, err := MockClient(t, server.Addr, cltMng)
	if err != nil {
		t.Errorf("request server error %v", err)
		return
	}
	defer resp.Body.Close()

	serverCN := resp.TLS.PeerCertificates[0].Subject.CommonName
	expected := testCases[0].Info.CommonName
	if serverCN != expected {
		t.Errorf("expected request server config  %s , but got %s", expected, serverCN)
	}

	ioutil.ReadAll(resp.Body)
}

func TestFallback(t *testing.T) {
	// tls config fallback is true. no certificate is configured
	// the server side trigger fallback, without any providers, so just support non-tls
	// the client side is normal config, with a empty provider, can send tls request
	t.Run("NoCertWithFallback", func(t *testing.T) {
		cfg := v2.TLSConfig{
			Status:   true,
			Fallback: true,
		}
		filterChains := []v2.FilterChain{
			{
				TLSContexts: []v2.TLSConfig{
					cfg,
				},
			},
		}
		lc := &v2.Listener{}
		lc.FilterChains = filterChains
		serverMgr, err := NewTLSServerContextManager(lc)
		if err != nil {
			t.Fatalf("create tls server context manager failed: %v", err)
		}
		if ctxMng, ok := serverMgr.(*serverContextManager); !ok || len(ctxMng.providers) != 0 {
			t.Error("server context manager have providers, but expected not")
		}
		clientMgr, err := NewTLSClientContextManager(&cfg)
		if err != nil {
			t.Fatalf("create tls client context manager failed: %v", err)
		}
		if ctxMng, ok := clientMgr.(*clientContextManager); !ok || !(ctxMng.provider != nil && ctxMng.provider.Empty()) {
			t.Error("clienr context manager expected a empty provider")
		}
	})
	// tls config fallback is true. invalid certificate is configured
	// the server side trigger fallback, without any providers, so just support non-tls
	// the client side is normal config, with a empty provider, can send tls request
	t.Run("InvalidCertWithFallback", func(t *testing.T) {
		cfg := v2.TLSConfig{
			Status:     true,
			CertChain:  "invalid_certificate",
			PrivateKey: "invalid_key",
			Fallback:   true,
		}
		filterChains := []v2.FilterChain{
			{
				TLSContexts: []v2.TLSConfig{
					cfg,
				},
			},
		}
		lc := &v2.Listener{}
		lc.FilterChains = filterChains
		serverMgr, err := NewTLSServerContextManager(lc)
		if err != nil {
			t.Fatalf("create tls server context manager failed: %v", err)
		}
		if ctxMng, ok := serverMgr.(*serverContextManager); !ok || len(ctxMng.providers) != 0 {
			t.Error("server context manager have providers, but expected not")
		}
		clientMgr, err := NewTLSClientContextManager(&cfg)
		if err != nil {
			t.Fatalf("create tls client context manager failed: %v", err)
		}
		if ctxMng, ok := clientMgr.(*clientContextManager); !ok || !(ctxMng.provider != nil && ctxMng.provider.Empty()) {
			t.Error("clienr context manager expected a empty provider")
		}
	})
	// an empty certificate is configured, without fallback
	// server tls context manager can not be created, but client side can be created
	t.Run("NoCertWithoutFallback", func(t *testing.T) {
		cfg := v2.TLSConfig{
			Status: true,
		}
		filterChains := []v2.FilterChain{
			{
				TLSContexts: []v2.TLSConfig{
					cfg,
				},
			},
		}
		lc := &v2.Listener{}
		lc.FilterChains = filterChains
		_, err := NewTLSServerContextManager(lc)
		if err == nil {
			t.Fatal("create tls server context without certificate success, expected failed")
		}
		clientMgr, err := NewTLSClientContextManager(&cfg)
		if err != nil {
			t.Fatalf("create tls client context manager failed: %v", err)
		}
		if ctxMng, ok := clientMgr.(*clientContextManager); !ok || !(ctxMng.provider != nil && ctxMng.provider.Empty()) {
			t.Error("clienr context manager expected a empty provider")
		}
	})
	// an inavlid certificate is configured, without fallback
	// no tls context manager can be created
	t.Run("InvalidCertWithoutFallback", func(t *testing.T) {
		cfg := v2.TLSConfig{
			Status:     true,
			CertChain:  "invalid_certificate",
			PrivateKey: "invalid_key",
		}
		filterChains := []v2.FilterChain{
			{
				TLSContexts: []v2.TLSConfig{
					cfg,
				},
			},
		}
		lc := &v2.Listener{}
		lc.FilterChains = filterChains
		if _, err := NewTLSServerContextManager(lc); err == nil {
			t.Fatal("create tls server context without certificate success, expected failed")
		}
		if _, err := NewTLSClientContextManager(&cfg); err == nil {
			t.Fatal("create tls client context without certificate success, expected failed")
		}
	})

}

func TestGmTLS(t *testing.T) {
	info := certInfo{"Cert1", "RSA", "www.example.com"}
	cfg, err := info.CreateCertConfig()
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	cfg.CipherSuites = "ECDHE-RSA-SM4-SM3"
	cfg.VerifyClient = false
	filterChains := []v2.FilterChain{
		{
			TLSContexts: []v2.TLSConfig{
				*cfg,
			},
		},
	}
	lc := &v2.Listener{}
	lc.Inspector = true
	lc.FilterChains = filterChains
	ctxMng, err := NewTLSServerContextManager(lc)
	if err != nil {
		t.Errorf("create context manager failed %v", err)
		return
	}
	server := MockServer{
		Mng: ctxMng,
		t:   t,
	}
	server.GoListenAndServe(t)
	defer server.Close()
	time.Sleep(time.Second) //wait server start

	clientConfig := v2.TLSConfig{
		Status:       true,
		CACert:       cfg.CACert,
		CertChain:    cfg.CertChain,
		PrivateKey:   cfg.PrivateKey,
		InsecureSkip: true,
		CipherSuites: "ECDHE-RSA-SM4-SM3",
	}
	cltMng, err := NewTLSClientContextManager(&clientConfig)
	if err != nil {
		t.Errorf("create client context manager failed %v", err)
	}

	resp, err := MockClient(t, server.Addr, cltMng)
	if err != nil {
		t.Errorf("request server error %v", err)
	}
	ioutil.ReadAll(resp.Body)
	resp.Body.Close()
}

func TestAlpnMatch(t *testing.T) {
	info := certInfo{"Cert1", "RSA", "www.example.com"}
	cfg, err := info.CreateCertConfig()
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	cfg.VerifyClient = false
	cfg.ALPN = "h2,http/1.1,sofa"
	filterChains := []v2.FilterChain{
		{
			TLSContexts: []v2.TLSConfig{
				*cfg,
			},
		},
	}
	lc := &v2.Listener{}
	lc.FilterChains = filterChains
	ctxMng, err := NewTLSServerContextManager(lc)
	if err != nil {
		t.Errorf("create context manager failed %v", err)
		return
	}
	server := MockServer{
		Mng: ctxMng,
		t:   t,
	}
	server.GoListenAndServe(t)
	defer server.Close()
	time.Sleep(time.Second) //wait server start

	clientConfig := v2.TLSConfig{
		Status:       true,
		CACert:       cfg.CACert,
		CertChain:    cfg.CertChain,
		PrivateKey:   cfg.PrivateKey,
		InsecureSkip: true,
		ALPN:         "sofa,http/1.1",
	}
	cltMng, err := NewTLSClientContextManager(&clientConfig)
	if err != nil {
		t.Errorf("create client context manager failed %v", err)
	}

	resp, err := MockClient(t, server.Addr, cltMng)
	if err != nil {
		t.Errorf("request server error %v", err)
	}
    //alpn select should follow server privilege
	alpnChoose := resp.TLS.NegotiatedProtocol 
	if alpnChoose != "http/1.1" {
		t.Errorf("alpn choose fail, want http/1.1, but %s", alpnChoose)
	}
	ioutil.ReadAll(resp.Body)
	resp.Body.Close()
}
