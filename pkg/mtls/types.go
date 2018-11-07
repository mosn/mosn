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

	"github.com/alipay/sofa-mosn/pkg/mtls/crypto/tls"
)

// Support Protocols version
const (
	minProtocols uint16 = tls.VersionTLS10
	maxProtocols uint16 = tls.VersionTLS12
)

// version string map
var version = map[string]uint16{
	"tls_auto": 0,
	"tlsv1_0":  tls.VersionTLS10,
	"tlsv1_1":  tls.VersionTLS11,
	"tlsv1_2":  tls.VersionTLS12,
}

// Curves
var (
	defaultCurves = []tls.CurveID{
		tls.X25519,
		tls.CurveP256,
	}
	allCurves = map[string]tls.CurveID{
		"x25519": tls.X25519,
		"p256":   tls.CurveP256,
		"p384":   tls.CurveP384,
		"p521":   tls.CurveP521,
	}
)

// ALPN
var alpn = map[string]bool{
	"h2":       true,
	"http/1.1": true,
	"sofa":     true,
}

// Ciphers
var (
	defaultCiphers = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
	}
	ciphersMap = map[string]uint16{
		"ECDHE-ECDSA-AES256-GCM-SHA384":      tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"ECDHE-RSA-AES256-GCM-SHA384":        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"ECDHE-ECDSA-AES128-GCM-SHA256":      tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"ECDHE-RSA-AES128-GCM-SHA256":        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"ECDHE-ECDSA-WITH-CHACHA20-POLY1305": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		"ECDHE-RSA-WITH-CHACHA20-POLY1305":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		"ECDHE-RSA-AES256-CBC-SHA":           tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"ECDHE-RSA-AES128-CBC-SHA":           tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"ECDHE-ECDSA-AES256-CBC-SHA":         tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"ECDHE-ECDSA-AES128-CBC-SHA":         tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"RSA-AES256-CBC-SHA":                 tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"RSA-AES128-CBC-SHA":                 tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"ECDHE-RSA-3DES-EDE-CBC-SHA":         tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		"RSA-3DES-EDE-CBC-SHA":               tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	}
)

// ConfigHooks is a  set of functions used to make a tls config
type ConfigHooks interface {
	// GetCertificate returns the tls.Certificate by index.
	// By default the index is the cert/key file path or cert/key pem string
	GetCertificate(certIndex, keyIndex string) (tls.Certificate, error)
	// GetX509Pool returns the x509.CertPool, which is a set of certificates.
	// By default the index is the ca certificate file path or certificate pem string
	GetX509Pool(caIndex string) (*x509.CertPool, error)
	// VerifyPeerCertificate returns a "VerifyPeerCertificate" defined in tls.Config.
	// If it is returns nil, the normal certificate verification will be used.
	// Notice that we set tls.Config.InsecureSkipVerify to make sure the "VerifyPeerCertificate" is called,
	// so an implement of "VerifyPeerCertificate" should verify the trusted ca if necessary.
	// If TLSConfig.InsecureSkip is true, the "VerifyPeerCertificate" will be ignored.
	VerifyPeerCertificate() func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error
}

// ConfigHooksFactory creates ConfigHooks by config
type ConfigHooksFactory interface {
	CreateConfigHooks(config map[string]interface{}) ConfigHooks
}

// ErrorNoCertConfigure represents config has no certificate
var ErrorNoCertConfigure = errors.New("no certificate config")

// ErrorGetCertificateFailed represents get certificate from config failed
var ErrorGetCertificateFailed = errors.New("get certificate failed")
