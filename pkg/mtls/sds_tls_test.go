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
	"fmt"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

var testInit sync.Once

func createSdsTLSConfig() *v2.TLSConfig {
	testInit.Do(func() {
		getSdsClientFunc = getMockSdsClient
	})
	return &v2.TLSConfig{
		Status:       true,
		VerifyClient: true,
		SdsConfig: &v2.SdsConfig{
			CertificateConfig: &v2.SecretConfigWrapper{
				Name: "default",
			},
			ValidationConfig: &v2.SecretConfigWrapper{
				Name: "rootCA",
			},
		},
	}
}

func resetTest() {
	secretManagerInstance = &secretManager{
		validations: make(map[string]*validation),
	}
	sdsCallbacks = map[string]func(*v2.TLSConfig){}
	mockSdsClientInstance = &mockSdsClient{
		callback: make(map[string]types.SdsUpdateCallbackFunc),
	}
}

func mockSetSecret() *SecretInfo {
	info := &certInfo{
		CommonName: "sds",
		Curve:      "RSA",
	}
	secret, _ := info.CreateSecret()
	//
	sRoot := &types.SdsSecret{
		Name:          "rootCA",
		ValidationPEM: secret.Validation,
	}
	sDefault := &types.SdsSecret{
		Name:           "default",
		CertificatePEM: secret.Certificate,
		PrivateKeyPEM:  secret.PrivateKey,
	}

	//
	mockSdsClientInstance.SetSecret("rootCA", sRoot)
	mockSdsClientInstance.SetSecret("default", sDefault)
	return secret
}

// Test a simple sds scenario
// server listen a sds tls config
// before the certificate is setted, cannot support tls request
// after the certificate is setted, support tls request
func TestSimpleSdsTLS(t *testing.T) {
	resetTest()
	cfg := createSdsTLSConfig()
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
	}
	server.GoListenAndServe()
	defer server.Close()
	time.Sleep(time.Second) //wait server start
	// before certificet set, server support non-tls
	inspectorCall := func() {
		resp, err := MockClient(server.Addr, nil)
		if err != nil {
			t.Errorf("inspector call sds listener failed: %v", err)
			return
		}
		defer resp.Body.Close()
		ioutil.ReadAll(resp.Body)
	}
	// no certificate in server, tls call should failed
	unsupportCall := func() {
		cfg := &v2.TLSConfig{
			Status:       true,
			InsecureSkip: true,
		}
		cltMng, err := NewTLSClientContextManager("", cfg)
		if err != nil {
			t.Fatalf("create tls context manager failed: %v", err)
		}
		resp, err := MockClient(server.Addr, cltMng)
		if err == nil {
			t.Error("expected request server failed, but success")
			ioutil.ReadAll(resp.Body)
			resp.Body.Close()
		}
	}
	inspectorCall()
	unsupportCall()
	secret := mockSetSecret()
	// after certificate set
	clientConfigs := []*v2.TLSConfig{
		{
			Status:     true,
			CACert:     secret.Validation,
			ServerName: "127.0.0.1",
		},
		{
			Status:       true,
			InsecureSkip: true,
		},
	}
	for i, cfg := range clientConfigs {
		cltMng, err := NewTLSClientContextManager("", cfg)
		if err != nil {
			t.Errorf("#%d create client context manager failed %v", i, err)
			continue
		}

		resp, err := MockClient(server.Addr, cltMng)
		if err != nil {
			t.Errorf("#%d request server error %v", i, err)
			continue
		}
		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}

}

// Test sds tls config with extension, after certificate is setted
// If the client request non-tls. it is ok
// If the client request tls without certificate. it is ok
// If the client request tls with certificate, the server will verify the client's certificate
func TestSdsWithExtension(t *testing.T) {
	resetTest()
	cfg := createSdsTLSConfig()
	// Add extension
	cfg.Type = testType
	extendVerify := map[string]interface{}{
		"name": "server",
		"cn":   "client",
	}
	cfg.ExtendVerify = extendVerify

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
	}
	server.GoListenAndServe()
	defer server.Close()
	time.Sleep(time.Second) //wait server start
	mockSetSecret()
	// 1. client request non-tls
	inspectorCall := func() {
		resp, err := MockClient(server.Addr, nil)
		if err != nil {
			t.Errorf("inspector call sds listener failed: %v", err)
			return
		}
		defer resp.Body.Close()
		ioutil.ReadAll(resp.Body)
	}
	inspectorCall()
	// 2. client request tls without certificate
	noCertCall := func() {
		cfg := &v2.TLSConfig{
			Status:       true,
			InsecureSkip: true,
		}
		cltMng, _ := NewTLSClientContextManager("", cfg)
		resp, err := MockClient(server.Addr, cltMng)
		if err != nil {
			t.Errorf("client request server without certificate failed: %v", err)
			return
		}
		defer resp.Body.Close()
		ioutil.ReadAll(resp.Body)

	}
	noCertCall()
	// 3. requets with valid cert, verify pass
	certPass := func() {
		info := &certInfo{
			CommonName: extendVerify["cn"].(string),
			Curve:      "RSA",
		}
		cfg, _ := info.CreateCertConfig()
		cfg.ServerName = "127.0.0.1"
		cltMng, err := NewTLSClientContextManager("", cfg)
		if err != nil {
			t.Errorf("create tls context manager failed: %v", err)
			return
		}
		resp, err := MockClient(server.Addr, cltMng)
		if err != nil {
			t.Errorf("client request server with valid certificate failed: %v", err)
			return
		}
		defer resp.Body.Close()
		ioutil.ReadAll(resp.Body)
	}
	certPass()
	// 4. request with invalid cert,  verify failed
	certFail := func() {
		info := &certInfo{
			CommonName: "invalid",
			Curve:      "RSA",
		}
		cfg, _ := info.CreateCertConfig()
		cfg.ServerName = "127.0.0.1"
		cltMng, err := NewTLSClientContextManager("", cfg)
		if err != nil {
			t.Errorf("create tls context manager failed: %v", err)
			return
		}
		resp, err := MockClient(server.Addr, cltMng)
		if err == nil {
			t.Errorf("client request server with invalid certificate passed")
			defer resp.Body.Close()
			ioutil.ReadAll(resp.Body)
		}
	}
	certFail()
}

func TestSdsProviderUpdate(t *testing.T) {
	resetTest()
	cfg := createSdsTLSConfig()
	prd := addOrUpdateProvider("test", cfg)
	if prd.Ready() {
		t.Fatal("provider ready without certificate")
	}
	mockSetSecret()
	if !prd.Ready() {
		t.Fatal("provider note ready with certificate")
	}
	h1 := prd.GetTLSConfigContext(true).HashValue()
	// try to get again
	prd2 := addOrUpdateProvider("test", cfg)
	prdAddr := fmt.Sprintf("%p", prd)
	prd2Addr := fmt.Sprintf("%p", prd2)
	if prdAddr != prd2Addr {
		t.Fatal("sds provider reuse failed")
	}
	// update tls config
	cfg2 := createSdsTLSConfig()
	cfg2.CipherSuites = "RSA-AES256-CBC-SHA:RSA-3DES-EDE-CBC-SHA"
	prd3 := addOrUpdateProvider("test", cfg2)
	prd3Addr := fmt.Sprintf("%p", prd3)
	if prdAddr != prd3Addr {
		t.Fatal("sds provider reuse failed")
	}
	h2 := prd.GetTLSConfigContext(true).HashValue()
	if h1 == h2 {
		t.Fatal("update tls config failed")
	}
}

func TestSecretManager(t *testing.T) {
	resetTest()
	cfg := createSdsTLSConfig()
	addOrUpdateProvider("test", cfg)
	mockSetSecret()
	require.Len(t, secretManagerInstance.validations, 1)
	vd, ok := secretManagerInstance.validations["rootCA"]
	require.True(t, ok)
	require.Len(t, vd.certificates, 1)
	pem, ok := vd.certificates["default"]
	require.True(t, ok)
	require.True(t, pem.rootca != "")
	require.True(t, pem.cert != "")
	require.True(t, pem.key != "")
	require.Len(t, pem.sdsProviders, 1)
}

// a sds config support to different tls config, different callbacks
func TestSdsCertForDifferentConfig(t *testing.T) {
	resetTest()
	// register a callback
	cbname := "test_callback"
	count := 0
	RegisterSdsCallback(cbname, func(cfg *v2.TLSConfig) {
		count++
		require.Equal(t, "test1", cfg.ServerName)
	})

	//
	cfg := createSdsTLSConfig()
	cfg.ServerName = "test1"
	cfg.Callbacks = []string{cbname}
	prd := addOrUpdateProvider("test1", cfg)
	mockSetSecret()
	cfg2 := createSdsTLSConfig()
	cfg2.ServerName = "test2"
	prd2 := addOrUpdateProvider("test2", cfg2)

	tlsConfig := prd.GetTLSConfigContext(true).Config()
	tlsConfig2 := prd2.GetTLSConfigContext(true).Config()

	require.True(t, prd.Ready())
	require.True(t, prd2.Ready())
	require.Equal(t, "test1", tlsConfig.ServerName)
	require.Equal(t, "test2", tlsConfig2.ServerName)
	require.Equal(t, 1, count)

}

func TestSdsWithoutValidation(t *testing.T) {
	resetTest()
	createSdsTLSConfig() // init

	cfg := &v2.TLSConfig{
		Status: true,
		SdsConfig: &v2.SdsConfig{
			CertificateConfig: &v2.SecretConfigWrapper{
				Name: "default",
			},
		},
	}
	prd := addOrUpdateProvider("test", cfg)
	require.False(t, prd.Ready())
	mockSetSecret()
	require.True(t, prd.Ready())
	tlsConfig := prd.GetTLSConfigContext(false).Config()
	require.Nil(t, tlsConfig.RootCAs)
	require.Nil(t, tlsConfig.ClientCAs)
}

func TestSdsCallbackConcur(t *testing.T) {
	// generate secret
	info := &certInfo{
		CommonName: "sds",
		Curve:      "RSA",
	}
	secret, _ := info.CreateSecret()
	sRoot := &types.SdsSecret{
		Name:          "rootCA",
		ValidationPEM: secret.Validation,
	}
	sDefault := &types.SdsSecret{
		Name:           "default",
		CertificatePEM: secret.Certificate,
		PrivateKeyPEM:  secret.PrivateKey,
	}
	// prepare for test
	indexes := make([]string, 100)
	for i := 0; i < 100; i++ {
		indexes[i] = fmt.Sprintf("index%d", i)
	}
	cfg := createSdsTLSConfig()
	log.DefaultLogger.SetLogLevel(log.ERROR) // ignore log

	for i := 0; i < 10; i++ {
		t.Logf("run case %d times", i+1)
		resetTest()
		// init a provider, make sure the set secret have callback to call
		addOrUpdateProvider("test", cfg)
		wg := sync.WaitGroup{}
		for j := 0; j < 100; j++ {
			wg.Add(3)
			// add new config
			go func(j int) {
				addOrUpdateProvider(indexes[j], cfg)
				wg.Done()
			}(j)
			// update config for provider
			go func() {
				addOrUpdateProvider("test", cfg)
				wg.Done()
			}()
			// set secret
			go func() {
				mockSdsClientInstance.SetSecret("rootCA", sRoot)
				mockSdsClientInstance.SetSecret("default", sDefault)
				wg.Done()
			}()
		}
		wg.Wait()
		// check, make sure expected
		require.Len(t, secretManagerInstance.validations, 1)
		vd, ok := secretManagerInstance.validations["rootCA"]
		require.True(t, ok)
		require.Len(t, vd.certificates, 1)
		pem, ok := vd.certificates["default"]
		require.True(t, ok)
		require.True(t, pem.rootca != "")
		require.True(t, pem.cert != "")
		require.True(t, pem.key != "")
		require.Len(t, pem.sdsProviders, 101)
		for _, sp := range pem.sdsProviders {
			require.True(t, sp.Ready())
		}
	}

}
