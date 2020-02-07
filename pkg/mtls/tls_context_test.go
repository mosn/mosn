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

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mtls/crypto/tls"
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
	if ctx.client.InsecureSkipVerify {
		t.Errorf("client certificate should verify the server")
	}
	if ctx.server.ClientAuth != tls.VerifyClientCertIfGiven {
		t.Errorf("server client auth is %v", ctx.server.ClientAuth)
	}
}

// a tls context with no certificate is used in client
func TestTLSContextNoCert(t *testing.T) {
	cfg := &v2.TLSConfig{
		Status:     true,
		ServerName: "127.0.0.1",
	}
	ctx, err := newTLSContext(cfg, &secretInfo{})
	if err != nil {
		t.Fatalf("create tls context failed, %v", err)
	}
	if ctx.GetTLSConfig(false) != nil {
		t.Error("context should not contains a server side config")
	}
	if ctx.GetTLSConfig(true) == nil {
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
