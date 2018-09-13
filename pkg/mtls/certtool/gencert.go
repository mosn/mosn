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

// Package certtool used for generate certificate for test/examples
// By default, use CreateTemplate, GeneratePrivateKey, and SignCertificate, the certificates created in same process have same root ca
package certtool

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"time"
)

type certificate struct {
	*CertificateInfo
	cert *x509.Certificate
	priv interface{}
}

// RootCA is created by Initialize().
// Certificates are signed by RootCA.
var rootCA *certificate

// GeneratePrivateKey generate a private key for certificate
// Parameter curve support "P224","P256","P384", "P521", "RSA".
// If curve is "RSA", it will generate a 2048 bits RSA key
func GeneratePrivateKey(curve string) (priv interface{}, err error) {
	switch curve {
	case "RSA":
		priv, err = rsa.GenerateKey(rand.Reader, 2048)
	case "P224":
		priv, err = ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	case "P256":
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "P384":
		priv, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "P521":
		priv, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	default:
		err = errors.New("unsupported curve type")
	}
	return
}

// PublicKey returns the private key's public key
func PublicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

// PemEncode encode the data to pem string
func PemEncode(describe string, data []byte) (string, error) {
	block := &pem.Block{
		Type:  describe,
		Bytes: data,
	}
	var buf bytes.Buffer
	err := pem.Encode(&buf, block)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// PemEncodeKey encode private key to pem string
func PemEncodeKey(priv interface{}) (string, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return PemEncode("RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(k))
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return "", err
		}
		return PemEncode("EC PRIVATE KEY", b)
	default:
		return "", errors.New("unknown private key")
	}
}

// CreateTemplate creates a basic template for certificate
func CreateTemplate(cn string, isca bool, dns []string) (*x509.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"sofa mesh tls util"},
			CommonName:   cn,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(100, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		DNSNames:              dns,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	if isca {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	} else {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	}
	return template, nil
}

// CreateCertificateInfo makes a CertificateInfo
func CreateCertificateInfo(template, parent *x509.Certificate, templatePriv, parentPriv interface{}) (*CertificateInfo, error) {
	cert, err := createCertificate(template, parent, templatePriv, parentPriv)
	if err != nil {
		return nil, err
	}
	return cert.CertificateInfo, nil
}

func createCertificate(template, parent *x509.Certificate, templatePriv, parentPriv interface{}) (*certificate, error) {
	der, err := x509.CreateCertificate(rand.Reader, template, parent, PublicKey(templatePriv), parentPriv)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	certPem, err := PemEncode("CERTIFICATE", der)
	if err != nil {
		return nil, err
	}
	privPem, err := PemEncodeKey(templatePriv)
	if err != nil {
		return nil, err
	}
	return &certificate{
		CertificateInfo: &CertificateInfo{
			CertPem: certPem,
			KeyPem:  privPem,
		},
		cert: cert,
		priv: templatePriv,
	}, nil

}

// Initialize creates the RootCA only if RootCA is nil
func Initialize() error {
	if rootCA != nil {
		return nil
	}
	ca, err := CreateTemplate("", true, nil)
	if err != nil {
		return err
	}
	priv, err := GeneratePrivateKey("RSA")
	if err != nil {
		return err
	}
	cert, err := createCertificate(ca, ca, priv, priv)
	if err != nil {
		return err
	}
	rootCA = cert
	return nil
}

// GetRootCA returns the RootCA's certificate info
func GetRootCA() *CertificateInfo {
	if rootCA == nil {
		if err := Initialize(); err != nil {
			return nil
		}
	}
	return &CertificateInfo{
		CertPem: rootCA.CertPem,
		KeyPem:  rootCA.KeyPem,
	}
}

// SignCertificate create a Certificate based on template, signed by RootCA
func SignCertificate(template *x509.Certificate, priv interface{}) (*CertificateInfo, error) {
	if rootCA == nil {
		if err := Initialize(); err != nil {
			return nil, err
		}
	}
	return CreateCertificateInfo(template, rootCA.cert, priv, rootCA.priv)
}
