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
	"testing"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mtls/crypto/tls"
	"mosn.io/mosn/pkg/types"
)

func TestTLSContext(t *testing.T) {
	info := &certInfo{
		CommonName: "test",
		Curve:      "P256",
		DNS:        "www.test.com",
	}
	secret, _ := info.CreateSecret()
	cfg := &v2.TLSConfig{
		Status:            true,
		CipherSuites:      "ECDHE-ECDSA-AES256-GCM-SHA384",
		EcdhCurves:        "p256",
		MaxVersion:        "tlsv1_2",
		MinVersion:        "tlsv1_0",
		ALPN:              "h2,http/1.1,sofa",
		ServerName:        "127.0.0.1",
		RequireClientCert: true,
	}
	ctx, err := newTLSContext(cfg, secret)
	if err != nil {
		t.Fatalf("create tls context failed, %v", err)
	}
	// verify
	expectedMatches := []string{
		"test",                   // common name
		"www.test.com",           // dns
		"h2", "http/1.1", "sofa", // next protos
		"127.0.0.1", // server name
	}
	for _, m := range expectedMatches {
		if _, ok := ctx.matches[m]; !ok {
			t.Errorf("%s not in context", m)
		}
	}
	if ctx.client.Config().InsecureSkipVerify {
		t.Errorf("client certificate should verify the server")
	}
	if ctx.server.Config().ClientAuth != tls.RequestClientCert {
		t.Errorf("server client auth is %v", ctx.server.Config().ClientAuth)
	}
}

// a tls context with no certificate is used in client
func TestTLSContextNoCert(t *testing.T) {
	cfg := &v2.TLSConfig{
		Status:     true,
		ServerName: "127.0.0.1",
	}
	ctx, err := newTLSContext(cfg, &SecretInfo{})
	if err != nil {
		t.Fatalf("create tls context failed, %v", err)
	}
	if ctx.GetTLSConfigContext(false) != nil {
		t.Error("context should not contains a server side config")
	}
	if ctx.GetTLSConfigContext(true) == nil {
		t.Error("context contains a client side config")
	}
}

func TestTLSContextMatch(t *testing.T) {
	info := &certInfo{
		DNS:   "*.test.com",
		Curve: "RSA",
	}
	secret, _ := info.CreateSecret()
	cfg := &v2.TLSConfig{
		Status: true,
		ALPN:   "h2",
	}
	ctx, err := newTLSContext(cfg, secret)
	if err != nil {
		t.Fatalf("create tls context failed, %v", err)
	}
	matchedSAN := []string{
		"www.test.com",
		"mail.test.com",
	}
	unmatchedSAN := []string{
		"www.test2.com",
		"test.com",
	}
	for _, san := range matchedSAN {
		if !ctx.MatchedServerName(san) {
			t.Errorf("%s matched failed", san)
		}
	}
	for _, san := range unmatchedSAN {
		if ctx.MatchedServerName(san) {
			t.Errorf("%s matched, but expected not", san)
		}
	}
	nextProtos := []string{"http/1.1", "h2"}
	if !ctx.MatchedALPN(nextProtos) {
		t.Error("protos matched failed")
	}

}

func TestTLSContextHash(t *testing.T) {
	info := &certInfo{
		CommonName: "test",
		Curve:      "RSA",
	}
	secret, _ := info.CreateSecret()
	cfg := &v2.TLSConfig{
		Status:            true,
		CipherSuites:      "ECDHE-ECDSA-AES256-GCM-SHA384:RSA-3DES-EDE-CBC-SHA",
		RequireClientCert: true,
	}
	createHash := func(cfg *v2.TLSConfig) (*types.HashValue, *types.HashValue) {
		ctx, err := newTLSContext(cfg, secret)
		if err != nil {
			t.Fatalf("create tls context failed, %v", err)
		}
		return ctx.GetTLSConfigContext(true).HashValue(),
			ctx.GetTLSConfigContext(false).HashValue()
	}
	c1, s1 := createHash(cfg)
	c2, s2 := createHash(cfg)
	if !(!c1.Equal(s1) &&
		c1.Equal(c2) &&
		s1.Equal(s2)) {
		t.Fatalf("config no changed, but hash value changed %s : %s, %s : %s \n", c1, c2, s1, s2)
	}
	// ClientAuth changed, client hash do not changed, server hash changed
	cfg2 := &v2.TLSConfig{
		Status:            true,
		CipherSuites:      "ECDHE-ECDSA-AES256-GCM-SHA384:RSA-3DES-EDE-CBC-SHA",
		RequireClientCert: true,
		VerifyClient:      true,
	}
	c3, s3 := createHash(cfg2)
	if !c3.Equal(c1) ||
		s3.Equal(s1) {
		t.Fatalf("hash value is not expected, %s : %s, %s : %s\n", c3, c1, s3, s1)
	}
	// CipherSuites changed, hash should be changed
	cfg3 := &v2.TLSConfig{
		Status:            true,
		CipherSuites:      "RSA-3DES-EDE-CBC-SHA:ECDHE-ECDSA-AES256-GCM-SHA384",
		RequireClientCert: true,
	}
	c4, s4 := createHash(cfg3)
	if c4.Equal(c1) || s4.Equal(s1) {
		t.Fatalf("hash value is not expected, %s : %s, %s : %s\n", c4, c1, s4, s1)
	}
}

func TestTLSContextHashNil(t *testing.T) {
	cfg := &v2.TLSConfig{
		Status: false,
	}
	cltMng, err := NewTLSClientContextManager("", cfg)
	if err != nil {
		t.Fatalf("new client manager failed: %v", err)
	}
	hash := cltMng.HashValue()
	if !hash.Equal(nil) {
		t.Fatalf("disabled client manager cannot returns a hash value")
	}
}
