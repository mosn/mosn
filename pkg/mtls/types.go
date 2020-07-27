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

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mtls/crypto/tls"
	"mosn.io/mosn/pkg/types"
)

// Support Protocols version
const (
	minProtocols uint16 = tls.VersionTLS10
	maxProtocols uint16 = tls.VersionTLS13
)

// version string map
var version = map[string]uint16{
	"tls_auto": 0,
	"tlsv1_0":  tls.VersionTLS10,
	"tlsv1_1":  tls.VersionTLS11,
	"tlsv1_2":  tls.VersionTLS12,
	"tlsv1_3":  tls.VersionTLS13,
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

// ConfigHooks is a  set of functions used to make a tls config
type ConfigHooks interface {
	// GetClientAuth sets the tls.Config's ClientAuth fields
	GetClientAuth(cfg *v2.TLSConfig) tls.ClientAuthType
	// GetCertificate returns the tls.Certificate by index.
	// By default the index is the cert/key file path or cert/key pem string
	GetCertificate(certIndex, keyIndex string) (tls.Certificate, error)
	// GetX509Pool returns the x509.CertPool, which is a set of certificates.
	// By default the index is the ca certificate file path or certificate pem string
	GetX509Pool(caIndex string) (*x509.CertPool, error)
	// ServerHandshakeVerify returns a function that used to set "VerifyPeerCertificate" defined in tls.Config.
	// If it is returns nil, the normal certificate verification will be used.
	// Notice that we set tls.Config.InsecureSkipVerify to make sure the "VerifyPeerCertificate" is called,
	// so the ServerHandshakeVerify should verify the trusted ca if necessary.
	// If the TLSConfig.RequireClientCert is false, the ServerHandshakeVerify will be ignored
	ServerHandshakeVerify(cfg *tls.Config) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error
	// ClientHandshakeVerify returns a function that used to set "VerifyPeerCertificate" defined in tls.Config.
	// If it is returns nil, the normal certificate verification will be used.
	// Notice that we set tls.Config.InsecureSkipVerify to make sure the "VerifyPeerCertificate" is called,
	// so the ClientHandshakeVerify should verify the trusted ca if necessary.
	// If TLSConfig.InsecureSkip is true, the ClientHandshakeVerify will be ignored.
	ClientHandshakeVerify(cfg *tls.Config) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error
	// GenerateHashValue creates a hash value based on the tls.Config
	GenerateHashValue(cfg *tls.Config) *types.HashValue
}

// ConfigHooksFactory creates ConfigHooks by config
type ConfigHooksFactory interface {
	CreateConfigHooks(config map[string]interface{}) ConfigHooks
}

// ErrorNoCertConfigure represents config has no certificate
var ErrorNoCertConfigure = errors.New("no certificate config")
