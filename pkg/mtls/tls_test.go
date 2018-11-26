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
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/mtls/certtool"
	"github.com/alipay/sofa-mosn/pkg/types"
	"golang.org/x/net/http2"
)

type MockListener struct {
	net.Listener
	Mng types.TLSContextManager
}

func MockClient(t *testing.T, addr string, cltMng types.TLSContextManager) (*http.Response, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("request server error %v", err)
	}
	var conn net.Conn
	var req *http.Request
	conn = c
	if cltMng != nil {
		req, _ = http.NewRequest("GET", "https://"+addr, nil)
		conn = cltMng.Conn(c)
		tlsConn, _ := conn.(*TLSConn)
		if err := tlsConn.Handshake(); err != nil {
			return nil, fmt.Errorf("request tls handshake error %v", err)
		}
	} else {
		req, _ = http.NewRequest("GET", "http://"+addr, nil)
	}

	transport := &http2.Transport{}
	h2Conn, err := transport.NewClientConn(conn)
	return h2Conn.RoundTrip(req)
}

func (ln MockListener) Accept() (net.Conn, error) {
	conn, err := ln.Listener.Accept()
	if err != nil {
		return conn, err
	}
	return ln.Mng.Conn(conn), nil
}

type MockServer struct {
	Mng      types.TLSContextManager
	Addr     string
	server   *http2.Server
	t        *testing.T
	listener *MockListener
}

func (s *MockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "mock server")
}

func (s *MockServer) GoListenAndServe(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Errorf("listen failed %v", err)
		return
	}
	s.Addr = ln.Addr().String()
	s.listener = &MockListener{ln, s.Mng}
	go s.serve(s.listener)
}

func (s *MockServer) serve(ln *MockListener) {
	for {
		c, e := ln.Accept()
		if e != nil {
			return
		}

		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/", s.ServeHTTP)
			server := &http2.Server{}
			s.server = server
			if tlsConn, ok := c.(*TLSConn); ok {
				tlsConn.SetALPN(http2.NextProtoTLS)
				if err := tlsConn.Handshake(); err != nil {
					s.t.Logf("Hanshake failed, %v", err)
					return
				}
			}
			server.ServeConn(c, &http2.ServeConnOpts{
				Handler: mux,
			})
		}()
	}
}

func (s *MockServer) Close() {
	s.listener.Close()
}

type certInfo struct {
	CommonName string
	Curve      string
	DNS        string
}

func (c *certInfo) CreateCertConfig() (*v2.TLSConfig, error) {
	priv, err := certtool.GeneratePrivateKey(c.Curve)
	if err != nil {
		return nil, fmt.Errorf("generate key failed %v", err)
	}
	var dns []string
	if c.DNS != "" {
		dns = append(dns, c.DNS)
	}
	tmpl, err := certtool.CreateTemplate(c.CommonName, false, dns)
	if err != nil {
		return nil, fmt.Errorf("generate certificate template failed %v", err)
	}
	cert, err := certtool.SignCertificate(tmpl, priv)
	if err != nil {
		return nil, fmt.Errorf("sign certificate failed %v", err)
	}
	return &v2.TLSConfig{
		Status:     true,
		CACert:     certtool.GetRootCA().CertPem,
		CertChain:  cert.CertPem,
		PrivateKey: cert.KeyPem,
	}, nil
}

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
			TLS: *cfg,
		}
		filterChains = append(filterChains, fc)
	}
	lc := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			FilterChains: filterChains,
		},
	}
	ctxMng, err := NewTLSServerContextManager(lc, nil, log.StartLogger)
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
		cltMng, err := NewTLSClientContextManager(cfg, nil)
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
	cltMng, err := NewTLSClientContextManager(cfg, nil)
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
	cfg.VerifyClient = true
	filterChains := []v2.FilterChain{
		{
			TLS: *cfg,
		},
	}
	lc := &v2.Listener{}
	lc.FilterChains = filterChains
	ctxMng, err := NewTLSServerContextManager(lc, nil, log.StartLogger)
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
		cltMng, err := NewTLSClientContextManager(cfg, nil)
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

	cltMng, err := NewTLSClientContextManager(cfg, nil)
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
			TLS: *cfg,
		},
	}
	lc := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Inspector:    true,
			FilterChains: filterChains,
		},
	}
	ctxMng, err := NewTLSServerContextManager(lc, nil, log.StartLogger)
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
	}, nil)
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

// test ConfigHooks
// define VerifyPeerCertificate, verify common name instead of san, ignore keyusage
type testConfigHooks struct {
	defaultConfigHooks
	Name           string
	Root           *x509.CertPool
	PassCommonName string
}

// over write
func (hook *testConfigHooks) VerifyPeerCertificate() func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return hook.verifyPeerCertificate
}
func (hook *testConfigHooks) GetX509Pool(caIndex string) (*x509.CertPool, error) {
	return hook.Root, nil
}

// verifiedChains is always nil
func (hook *testConfigHooks) verifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	var certs []*x509.Certificate
	for _, asn1Data := range rawCerts {
		cert, err := x509.ParseCertificate(asn1Data)
		if err != nil {
			return err
		}
		certs = append(certs, cert)
	}
	intermediates := x509.NewCertPool()
	for _, cert := range certs[1:] {
		intermediates.AddCert(cert)
	}
	opts := x509.VerifyOptions{
		Roots:         hook.Root,
		Intermediates: intermediates,
	}
	leaf := certs[0]
	_, err := leaf.Verify(opts)
	if err != nil {
		return err
	}
	if leaf.Subject.CommonName != hook.PassCommonName {
		return errors.New("common name miss match")
	}
	return nil
}

func pass(resp *http.Response, err error) bool {
	if err != nil {
		return false
	}
	ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return true
}
func fail(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return false
}

const testType = "test"

type testConfigHooksFactory struct{}

func (f *testConfigHooksFactory) CreateConfigHooks(config map[string]interface{}) ConfigHooks {
	c := make(map[string]string)
	for k, v := range config {
		if s, ok := v.(string); ok {
			c[strings.ToLower(k)] = s
		}
	}
	root := certtool.GetRootCA()
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM([]byte(root.CertPem))
	return &testConfigHooks{
		defaultConfigHooks: defaultConfigHooks{},
		Name:               c["name"],
		PassCommonName:     c["cn"],
		Root:               pool,
	}
}

// TestTLSExtensionsVerifyClient tests server allow request with certificate's common name is client only
func TestTLSExtensionsVerifyClient(t *testing.T) {
	// Server
	extendVerify := map[string]interface{}{
		"name": "server",
		"cn":   "client",
	}
	serverInfo := &certInfo{
		CommonName: extendVerify["name"].(string),
		Curve:      "RSA",
	}
	serverConfig, err := serverInfo.CreateCertConfig()
	if err != nil {
		t.Errorf("create server certificate error %v", err)
		return
	}
	serverConfig.VerifyClient = true
	serverConfig.Type = testType
	serverConfig.ExtendVerify = extendVerify
	filterChains := []v2.FilterChain{
		{
			TLS: *serverConfig,
		},
	}
	lc := &v2.Listener{}
	lc.FilterChains = filterChains
	ctxMng, err := NewTLSServerContextManager(lc, nil, log.StartLogger)
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
	testCases := []struct {
		Info *certInfo
		Pass func(resp *http.Response, err error) bool
	}{
		{
			Info: &certInfo{
				CommonName: extendVerify["cn"].(string),
				Curve:      serverInfo.Curve,
			},
			Pass: pass,
		},
		{
			Info: &certInfo{
				CommonName: "invalid client",
				Curve:      serverInfo.Curve,
			},
			Pass: fail,
		},
	}
	for i, tc := range testCases {
		cfg, err := tc.Info.CreateCertConfig()
		cfg.ServerName = "127.0.0.1"
		if err != nil {
			t.Errorf("#%d create client certificate error %v", i, err)
			continue
		}
		cltMng, err := NewTLSClientContextManager(cfg, nil)
		if err != nil {
			t.Errorf("#%d create client context manager failed %v", i, err)
			continue
		}

		resp, err := MockClient(t, server.Addr, cltMng)
		if !tc.Pass(resp, err) {
			t.Errorf("#%d verify failed", i)
		}
	}
}

// TestTestTLSExtensionsVerifyServer tests client accept server response with cerificate's common name is server only
func TestTestTLSExtensionsVerifyServer(t *testing.T) {
	extendVerify := map[string]interface{}{
		"name": "client",
		"cn":   "server",
	}
	clientInfo := &certInfo{
		CommonName: extendVerify["name"].(string),
		Curve:      "RSA",
	}

	testCases := []struct {
		Info *certInfo
		Pass func(resp *http.Response, err error) bool
	}{
		{
			Info: &certInfo{
				CommonName: extendVerify["cn"].(string),
				Curve:      clientInfo.Curve,
				DNS:        "www.pass.com",
			},
			Pass: pass,
		},
		{
			Info: &certInfo{
				CommonName: "invalid server",
				Curve:      clientInfo.Curve,
				DNS:        "www.fail.com",
			},
			Pass: fail,
		},
	}
	var filterChains []v2.FilterChain
	for i, tc := range testCases {
		cfg, err := tc.Info.CreateCertConfig()
		if err != nil {
			t.Errorf("#%d %v", i, err)
			return
		}
		fc := v2.FilterChain{
			TLS: *cfg,
		}
		filterChains = append(filterChains, fc)
	}
	lc := &v2.Listener{}
	lc.FilterChains = filterChains
	ctxMng, err := NewTLSServerContextManager(lc, nil, log.StartLogger)
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
	clientConfig, err := clientInfo.CreateCertConfig()
	if err != nil {
		t.Errorf("create client certificate error %v", err)
		return
	}
	clientConfig.Type = testType
	clientConfig.ExtendVerify = extendVerify
	for i, tc := range testCases {
		clientConfig.ServerName = tc.Info.DNS
		cltMng, err := NewTLSClientContextManager(clientConfig, nil)
		if err != nil {
			t.Errorf("create client context manager failed %v", err)
			return
		}

		resp, err := MockClient(t, server.Addr, cltMng)
		if !tc.Pass(resp, err) {
			t.Errorf("#%d verify failed", i)
		}
	}
	// insecure skip will skip even if it is registered
	skipConfig := &v2.TLSConfig{
		Status:       true,
		Type:         clientConfig.Type,
		CACert:       clientConfig.CACert,
		CertChain:    clientConfig.CertChain,
		PrivateKey:   clientConfig.PrivateKey,
		InsecureSkip: true,
	}
	for i, tc := range testCases {
		skipConfig.ServerName = tc.Info.DNS
		skipMng, err := NewTLSClientContextManager(skipConfig, nil)
		if err != nil {
			t.Errorf("create client context manager failed %v", err)
			return
		}
		resp, err := MockClient(t, server.Addr, skipMng)
		// ignore the case, must be pass
		if !pass(resp, err) {
			t.Errorf("#%d skip verify failed", i)
		}
	}
}

func TestFallback(t *testing.T) {
	cfg := v2.TLSConfig{
		Status:     true,
		CertChain:  "invalid_certificate",
		PrivateKey: "invalid_key",
		Fallback:   true,
	}
	filterChains := []v2.FilterChain{
		{
			TLS: cfg,
		},
	}
	lc := &v2.Listener{}
	lc.FilterChains = filterChains
	serverMgr, err := NewTLSServerContextManager(lc, nil, log.StartLogger)
	if err != nil {
		t.Errorf("create context manager failed %v", err)
		return
	}
	if serverMgr.Enabled() {
		t.Error("tls maanger is not fallabck")
		return
	}
	clientMgr, err := NewTLSClientContextManager(&cfg, nil)
	if err != nil {
		t.Errorf("create client context manager failed %v", err)
		return
	}
	if clientMgr.Enabled() {
		t.Error("tls maanger is not fallabck")
		return
	}
}

func TestMain(m *testing.M) {
	Register(testType, &testConfigHooksFactory{})
	m.Run()
}
