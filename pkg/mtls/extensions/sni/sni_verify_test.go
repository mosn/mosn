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

package sni

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/mtls/certtool"
)

const test_uri = "https://mosn.io/test"

type secretInfo struct {
	Certificate string
	PrivateKey  string
	Validation  string
}

// create a x509.Certificate with URIs, and encode it to pem
func createURICert() (*secretInfo, error) {
	priv, err := certtool.GeneratePrivateKey("RSA")
	if err != nil {
		return nil, err
	}
	tmpl, err := certtool.CreateTemplate("mosn test", false, nil)
	if err != nil {
		return nil, err
	}
	// change to uri mode
	tmpl.IPAddresses = nil
	uri, _ := url.Parse(test_uri)
	tmpl.URIs = []*url.URL{uri}
	cert, err := certtool.SignCertificate(tmpl, priv)
	if err != nil {
		return nil, err
	}
	return &secretInfo{
		Certificate: cert.CertPem,
		PrivateKey:  cert.KeyPem,
		Validation:  certtool.GetRootCA().CertPem,
	}, nil
}

func TestSniVerify(t *testing.T) {
	si, err := createURICert()
	require.Nil(t, err)
	// create tls config for server
	serverCfg := v2.TLSConfig{
		Status:     true,
		CACert:     si.Validation,
		CertChain:  si.Certificate,
		PrivateKey: si.PrivateKey,
	}
	filterChains := []v2.FilterChain{
		{
			TLSContexts: []v2.TLSConfig{
				serverCfg,
			},
		},
	}
	lc := &v2.Listener{}
	lc.FilterChains = filterChains
	ctxMng, err := mtls.NewTLSServerContextManager(lc)
	require.Nil(t, err)
	// start a mock server
	server := mtls.MockServer{
		Mng: ctxMng,
	}
	server.GoListenAndServe()
	defer server.Close()
	time.Sleep(time.Second) //wait server start
	t.Run("check uri success", func(t *testing.T) {
		// create tls config for client, should verify server uris
		clientCfg_paas := &v2.TLSConfig{
			Status: true,
			CACert: si.Validation,
			Type:   SniVerify,
			ExtendVerify: map[string]interface{}{
				ConfigKey: test_uri,
			},
		}
		cltMng, err := mtls.NewTLSClientContextManager("", clientCfg_paas)
		require.Nil(t, err)
		resp, err := mtls.MockClient(server.Addr, cltMng)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		// release sources
		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	})
	t.Run("check uri failed", func(t *testing.T) {
		// create a tls config for client, verify another uris, expected failed
		clientCfg_failed := &v2.TLSConfig{
			Status: true,
			CACert: si.Validation,
			Type:   SniVerify,
			ExtendVerify: map[string]interface{}{
				ConfigKey: "https://mosn.io/another",
			},
		}
		cltMng, err := mtls.NewTLSClientContextManager("", clientCfg_failed)
		require.Nil(t, err)
		_, err = mtls.MockClient(server.Addr, cltMng)
		require.NotNil(t, err)
	})
}
