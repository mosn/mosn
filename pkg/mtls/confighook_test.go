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
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mtls/certtool"
	"mosn.io/mosn/pkg/mtls/crypto/tls"
)

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

// test ConfigHooks
// define VerifyPeerCertificate, verify common name instead of san, ignore keyusage
type testConfigHooks struct {
	defaultConfigHooks
	Name           string
	PassCommonName string
}

// over write
func (hook *testConfigHooks) GetX509Pool(caIndex string) (*x509.CertPool, error) {
	// usually the certpool should make by caIndex
	root := certtool.GetRootCA()
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM([]byte(root.CertPem))
	return pool, nil
}

func (hook *testConfigHooks) verifyPeerCertificate(roots *x509.CertPool, certs []*x509.Certificate, t time.Time) error {
	opts := x509.VerifyOptions{
		Roots:         roots,
		CurrentTime:   t,
		Intermediates: x509.NewCertPool(),
	}
	for _, cert := range certs[1:] {
		opts.Intermediates.AddCert(cert)
	}
	leaf := certs[0]
	_, err := leaf.Verify(opts)
	if err != nil {
		return err
	}
	if leaf.Subject.CommonName != hook.PassCommonName {
		return errors.New("tls: common name miss match")
	}
	return nil
}

func (hook *testConfigHooks) ServerHandshakeVerify(cfg *tls.Config) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		certs := make([]*x509.Certificate, 0, len(rawCerts))
		for _, asn1Data := range rawCerts {
			cert, err := x509.ParseCertificate(asn1Data)
			if err != nil {
				return err
			}
			certs = append(certs, cert)
		}
		// If cfg.ClientAuth is tls.RequireAndVerifyClientCert and len(certs) == 0
		// it will return error before call the verifyPeerCertificate
		if cfg.ClientAuth >= tls.VerifyClientCertIfGiven && len(certs) > 0 {
			var t time.Time
			if cfg.Time != nil {
				t = cfg.Time()
			} else {
				t = time.Now()
			}
			return hook.verifyPeerCertificate(cfg.ClientCAs, certs, t)
		}
		return nil
	}
}

func (hook *testConfigHooks) ClientHandshakeVerify(cfg *tls.Config) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	// if cfg.InsecureSkipVerify is true, the function should never be called
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		certs := make([]*x509.Certificate, 0, len(rawCerts))
		for _, asn1Data := range rawCerts {
			cert, err := x509.ParseCertificate(asn1Data)
			if err != nil {
				return err
			}
			certs = append(certs, cert)
		}
		var t time.Time
		if cfg.Time != nil {
			t = cfg.Time()
		} else {
			t = time.Now()
		}
		return hook.verifyPeerCertificate(cfg.RootCAs, certs, t)

	}
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
	return &testConfigHooks{
		defaultConfigHooks: defaultConfigHooks{},
		Name:               c["name"],
		PassCommonName:     c["cn"],
	}
}

func TestMain(m *testing.M) {
	Register(testType, &testConfigHooksFactory{})
	os.Exit(m.Run())
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
	serverConfig.RequireClientCert = true
	serverConfig.VerifyClient = true
	serverConfig.Type = testType
	serverConfig.ExtendVerify = extendVerify
	filterChains := []v2.FilterChain{
		{
			TLSContexts: []v2.TLSConfig{
				*serverConfig,
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
	}
	server.GoListenAndServe()
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
		cltMng, err := NewTLSClientContextManager("", cfg)
		if err != nil {
			t.Errorf("#%d create client context manager failed %v", i, err)
			continue
		}

		resp, err := MockClient(server.Addr, cltMng)
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
			TLSContexts: []v2.TLSConfig{
				*cfg,
			},
		}
		filterChains = append(filterChains, fc)
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
	}
	server.GoListenAndServe()
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
		cltMng, err := NewTLSClientContextManager("", clientConfig)
		if err != nil {
			t.Errorf("create client context manager failed %v", err)
			return
		}

		resp, err := MockClient(server.Addr, cltMng)
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
		skipMng, err := NewTLSClientContextManager("", skipConfig)
		if err != nil {
			t.Errorf("create client context manager failed %v", err)
			return
		}
		resp, err := MockClient(server.Addr, skipMng)
		// ignore the case, must be pass
		if !pass(resp, err) {
			t.Errorf("#%d skip verify failed", i)
		}
	}

}
