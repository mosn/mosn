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
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/mtls/crypto/tls"
)

const (
	SniVerify = "sni_verify"
	ConfigKey = "match_subject_alt_names"
)

// SniVerifyHooksFactory is an implementation of ConfigHooksFactory
type SniVerifyHooksFactory struct{}

func init() {
	mtls.Register(SniVerify, &SniVerifyHooksFactory{})
}

func (f *SniVerifyHooksFactory) CreateConfigHooks(config map[string]interface{}) mtls.ConfigHooks {
	var uri string
	if v, ok := config[ConfigKey]; ok {
		if s, yes := v.(string); yes {
			uri = s
		}
	}
	// if uri is empty, fallback to default action
	if uri == "" {
		return mtls.DefaultConfigHooks()
	}
	return &sniVerifyHooks{
		ConfigHooks: mtls.DefaultConfigHooks(),
		uri:         uri,
	}
}

// sniVerifyHooks is a tls extensions that used to verify server URI
type sniVerifyHooks struct {
	mtls.ConfigHooks
	uri string
}

// ClientHandshakeVerify verify the cert.URIs.
func (hook *sniVerifyHooks) ClientHandshakeVerify(cfg *tls.Config) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		return hook.clientVerifyPeerCertificate(rawCerts, cfg)
	}
}

// clientVerifyPeerCertificate verify the cert.URIs instead of DNSNames
func (hook *sniVerifyHooks) clientVerifyPeerCertificate(rawCerts [][]byte, cfg *tls.Config) error {
	// len(rawCerts) is always > 0, or this function will never be called. see details in: crypto/tls: doFullHandshake
	certs := make([]*x509.Certificate, 0, len(rawCerts))
	for _, asn1Data := range rawCerts {
		cert, err := tls.LoadOrStoreCertificate(asn1Data)
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
	// verify the cert without DNSNames first, if failed, returns an error.
	opts := x509.VerifyOptions{
		Roots:         cfg.RootCAs,
		CurrentTime:   t,
		Intermediates: x509.NewCertPool(),
	}
	for _, cert := range certs[1:] {
		opts.Intermediates.AddCert(cert)
	}
	leaf := certs[0]
	if _, err := leaf.Verify(opts); err != nil {
		return err
	}
	// verify the servre URIs
	// one of the uris matched is ok
	for _, uri := range leaf.URIs {
		if strings.EqualFold(uri.String(), hook.uri) {
			return nil
		}
	}
	return fmt.Errorf("tls: SAN uri not matched, want: %s, cert URIs: %v", hook.uri, leaf.URIs)
}
