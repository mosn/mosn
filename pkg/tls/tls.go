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
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"net"

	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var TLSdefaultMinProtocols uint16 = tls.VersionTLS10
var TLSdefaultMaxProtocols uint16 = tls.VersionTLS12

var TLSProtocols = map[string]uint16{
	"tls_auto": 0,
	"tlsv1_0":  tls.VersionTLS10,
	"tlsv1_1":  tls.VersionTLS11,
	"tlsv1_2":  tls.VersionTLS12,
}

var TLSdefaultCurves = []tls.CurveID{
	tls.X25519,
	tls.CurveP256,
}

var TLSALPN = map[string]bool{
	"h2":       true,
	"http/1.1": true,
	"sofa":     true,
}

var TLSCurves = map[string]tls.CurveID{
	"x25519": tls.X25519,
	"p256":   tls.CurveP256,
	"p384":   tls.CurveP384,
	"p521":   tls.CurveP521,
}

var TLSdefaultCiphers = []uint16{
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

var TLSCiphersMap = map[string]uint16{
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

type context struct {
	serverName   string
	minVersion   uint16
	maxVersion   uint16
	alpn         []string
	clientAuth   tls.ClientAuthType
	caCert       *x509.CertPool
	cipherSuites []uint16
	ecdhCurves   []tls.CurveID
	certificates []tls.Certificate
	ticket       string

	verifyClient bool
	verifyServer bool

	listener    types.Listener
	clusterInfo types.ClusterInfo

	tlsConfig *tls.Config
}

type contextManager struct {
	contextMap []map[string]*context

	logger     log.Logger
	isClient   bool
	inspector  bool
	tlscontext *context
	listener   types.Listener

	sync.RWMutex
}

func buildContextMap(cm *contextManager, tlscontext *context, index int) {
	if tlscontext == nil {
		if index >= len(cm.contextMap) {
			cm.contextMap = append(cm.contextMap, nil)
		} else {
			cm.contextMap[index] = nil
		}
		return
	}

	c := tlscontext.tlsConfig

	m := make(map[string]*context)

	for i := range c.Certificates {
		cert := &c.Certificates[i]
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			continue
		}
		if len(x509Cert.Subject.CommonName) > 0 {
			m[x509Cert.Subject.CommonName] = tlscontext
		}
		for _, san := range x509Cert.DNSNames {
			m[san] = tlscontext
		}
	}

	if tlscontext.alpn != nil {
		for _, protocol := range tlscontext.alpn {
			m[protocol] = tlscontext
		}
	}

	m[tlscontext.serverName] = tlscontext

	if index >= len(cm.contextMap) {
		cm.contextMap = append(cm.contextMap, m)
	} else {
		cm.contextMap[index] = m
	}
}

func NewTLSServerContextManager(config []v2.FilterChain, l types.Listener, logger log.Logger) types.TLSContextManager {
	cm := new(contextManager)
	cm.logger = logger

	first := true
	for i, c := range config {
		tlscontext, err := newTLSContext(&c.TLS, cm)
		if err != nil {
			cm.logger.Errorf("New Server TLS Context Manager failed: %v", err)
			return nil
		}

		buildContextMap(cm, tlscontext, i)

		if tlscontext != nil {
			tlscontext.listener = l
		}

		if first && tlscontext != nil {
			cm.tlscontext = tlscontext
			first = false
		}
	}

	return cm
}

func AddTLSServerContext(c *v2.TLSConfig, tlsMng types.TLSContextManager, index int) error {
	var cm *contextManager

	if tlsMng == nil {
		return errors.New("Add Server TLS Context failed: tlsMng is nil")
	}

	if c, ok := tlsMng.(*contextManager); ok {
		cm = c
	} else {
		return errors.New("Add Server TLS Context failed: tlsMng is not tls.ContextManager")
	}

	if index < 0 || index > len(cm.contextMap) {
		return errors.New("Add Server TLS Context failed: index is out of bounds")
	}

	tlscontext, err := newTLSContext(c, cm)
	if err != nil {
		return err
	}

	tlscontext.listener = cm.listener

	//todo sync.RWMutex, Maps are not safe for concurrent use
	buildContextMap(cm, tlscontext, index)

	if cm.tlscontext == nil {
		cm.tlscontext = tlscontext
	}

	return nil
}

func DelTLSServerContext(tlsMng types.TLSContextManager, index int) error {
	var cm *contextManager
	if tlsMng == nil {
		return errors.New("Del Server TLS Context failed: tlsMng is nil")
	}

	if c, ok := tlsMng.(*contextManager); ok {
		cm = c
	} else {
		return errors.New("Del Server TLS Context failed: tlsMng is not tls.ContextManager")
	}

	if index < 0 || index >= len(cm.contextMap) {
		return errors.New("Del Server TLS Context failed: index is out of bounds")
	}

	maps := make([]map[string]*context, 0)
	for i, m := range cm.contextMap {
		if i != index {
			maps = append(maps, m)
		}
	}

	cm.contextMap = maps

	return nil
}

func NewTLSClientContextManager(config *v2.TLSConfig, info types.ClusterInfo) types.TLSContextManager {
	cm := new(contextManager)
	cm.isClient = true
	cm.logger = log.DefaultLogger

	tlscontext, err := newTLSContext(config, cm)
	if err != nil {
		cm.logger.Errorf("New TLS Client Context Manager failed: %v", err)
		return nil
	}

	if tlscontext == nil {
		return cm
	}

	tlscontext.clusterInfo = info
	cm.tlscontext = tlscontext

	return cm
}

func (cm *contextManager) GetConfigForClient(info *tls.ClientHelloInfo) (*tls.Config, error) {
	var tlscontext *context
	var ok bool

	// only one Cartificate
	if len(cm.contextMap) == 1 {
		return cm.tlscontext.tlsConfig.Clone(), nil
	}

	for _, maps := range cm.contextMap {
		if maps == nil {
			continue
		}
		// first match ServerName
		// e.g. www.example.com will be first matched against www.example.com, then *.example.com, then *.com
		if info.ServerName != "" {
			name := strings.ToLower(info.ServerName)
			for len(name) > 0 && name[len(name)-1] == '.' {
				name = name[:len(name)-1]
			}

			if tlscontext, ok = maps[name]; ok {
				goto find
			}

			labels := strings.Split(name, ".")
			for i := 0; i < len(labels)-1; i++ {
				labels[i] = "*"
				candidate := strings.Join(labels[i:], ".")
				if tlscontext, ok = maps[candidate]; ok {
					goto find
				}
			}
		}

		// Sencond match ALPN
		if info.SupportedProtos != nil {
			for _, protocol := range info.SupportedProtos {
				protocol = strings.ToLower(protocol)
				if tlscontext, ok = maps[protocol]; ok {
					goto find
				}
			}
		}
	}

	// Last, return the first certificate.
	tlscontext = cm.tlscontext

find:

	// todo
	// callback select filter config
	// callback(cm.listener, index)

	return tlscontext.tlsConfig.Clone(), nil

}

func (cm *contextManager) Enabled() bool {
	return cm.defaultContext() != nil
}

func (cm *contextManager) Conn(c net.Conn) net.Conn {
	tlscontext := cm.defaultContext()

	if cm.isClient {
		return tls.Client(c, tlscontext.tlsConfig)
	}

	if !cm.inspector {
		return tls.Server(c, tlscontext.tlsConfig)
	}

	tlsconn := &conn{
		Conn: c,
	}

	buf := tlsconn.Peek()
	if buf == nil {
		return tls.Server(tlsconn, tlscontext.tlsConfig)
	}

	switch buf[0] {
	// TLS handshake
	case 0x16:
		return tls.Server(tlsconn, tlscontext.tlsConfig)
	// http plain
	default:
		return tlsconn
	}
}

func (cm *contextManager) defaultContext() *context {
	return cm.tlscontext
}

func (c *context) newTLSConfig(cm *contextManager) error {
	config := new(tls.Config)
	config.Certificates = c.certificates
	config.GetConfigForClient = cm.GetConfigForClient
	config.MaxVersion = c.maxVersion
	config.MinVersion = c.minVersion
	config.CipherSuites = c.cipherSuites
	config.CurvePreferences = c.ecdhCurves
	config.NextProtos = c.alpn
	config.InsecureSkipVerify = true

	if c.verifyClient {
		config.ClientCAs = c.caCert
		config.ClientAuth = tls.RequireAndVerifyClientCert
	}

	if c.verifyServer {
		config.RootCAs = c.caCert
		config.InsecureSkipVerify = false
	}

	if cm.isClient {
		config.ServerName = c.serverName
	}

	c.tlsConfig = config

	return nil
}

func newTLSContext(c *v2.TLSConfig, cm *contextManager) (*context, error) {
	if c.Status == false {
		return nil, nil
	}

	tlscontext := new(context)

	if c.CipherSuites != "" {
		ciphers := strings.Split(c.CipherSuites, ":")
		for _, s := range ciphers {
			cipher, ok := TLSCiphersMap[s]
			if !ok {
				return nil, errors.New("error cipher name or cipher not supported")
			}

			tlscontext.cipherSuites = append(tlscontext.cipherSuites, cipher)
		}

		if len(tlscontext.cipherSuites) == 0 {
			tlscontext.cipherSuites = TLSdefaultCiphers
		}
	}

	if c.EcdhCurves != "" {
		curves := strings.Split(c.EcdhCurves, ",")
		for _, s := range curves {
			curve, ok := TLSCurves[strings.ToLower(s)]
			if !ok {
				return nil, errors.New("error curve name or curve not supported")
			}

			tlscontext.ecdhCurves = append(tlscontext.ecdhCurves, curve)
		}

		if len(tlscontext.ecdhCurves) == 0 {
			tlscontext.ecdhCurves = TLSdefaultCurves
		}
	}

	if c.MaxVersion != "" {
		protocol, ok := TLSProtocols[strings.ToLower(c.MaxVersion)]
		if !ok {
			return nil, errors.New("error tls protocols name or protocol not supported")
		}

		if protocol == 0 {
			tlscontext.maxVersion = TLSdefaultMaxProtocols

		} else {
			tlscontext.maxVersion = protocol
		}
	} else {
		tlscontext.maxVersion = TLSdefaultMaxProtocols
	}

	if c.MinVersion != "" {
		protocol, ok := TLSProtocols[strings.ToLower(c.MinVersion)]
		if !ok {
			return nil, errors.New("error tls protocols name or protocol not supported")
		}

		if protocol == 0 {
			tlscontext.minVersion = TLSdefaultMinProtocols

		} else {
			tlscontext.minVersion = protocol
		}
	} else {
		tlscontext.minVersion = TLSdefaultMinProtocols
	}

	if c.ALPN != "" {
		protocols := strings.Split(c.ALPN, ",")
		for _, p := range protocols {
			_, ok := TLSALPN[strings.ToLower(p)]
			if !ok {
				return nil, errors.New("error ALPN or ALPN not supported")
			}
			tlscontext.alpn = append(tlscontext.alpn, p)
		}
	}

	if c.CertChain != "" && c.PrivateKey != "" {
		var cert tls.Certificate
		var err error

		if strings.Contains(c.CertChain, "-----BEGIN") &&
			strings.Contains(c.PrivateKey, "-----BEGIN") {
			cert, err = tls.X509KeyPair([]byte(c.CertChain), []byte(c.PrivateKey))
		} else {
			cert, err = tls.LoadX509KeyPair(c.CertChain, c.PrivateKey)
		}
		if err != nil {
			return nil, fmt.Errorf("load [certchain] or [privatekey] error: %v", err)
		}

		tlscontext.certificates = append(tlscontext.certificates, cert)
	}

	if !cm.isClient && len(tlscontext.certificates) == 0 {
		return nil, errors.New("[certchain] and [privatekey] are required in TLS config")
	}

	if c.CACert != "" {
		var err error
		var ca []byte

		if strings.Contains(c.CACert, "-----BEGIN") {
			ca = []byte(c.CACert)
		} else {
			ca, err = ioutil.ReadFile(c.CACert)
			if err != nil {
				return nil, fmt.Errorf("load [cacert] error: %v", err)
			}
		}

		pool := x509.NewCertPool()
		if ok := pool.AppendCertsFromPEM(ca); !ok {
			return nil, errors.New("parse [cacert] error")
		}

		tlscontext.caCert = pool
	}

	tlscontext.verifyClient = c.VerifyClient
	tlscontext.verifyServer = c.VerifyServer

	tlscontext.serverName = c.ServerName

	if c.Inspector {
		cm.inspector = true
	}

	if err := tlscontext.newTLSConfig(cm); err != nil {
		return nil, err
	}

	return tlscontext, nil
}

type conn struct {
	net.Conn
	peek    [1]byte
	haspeek bool
}

func (c *conn) Peek() []byte {
	b := make([]byte, 1, 1)
	n, err := c.Conn.Read(b)
	if n == 0 {
		log.DefaultLogger.Infof("TLS Peek() error: %v", err)
		return nil
	}

	c.peek[0] = b[0]
	c.haspeek = true
	return b
}

func (c *conn) Read(b []byte) (int, error) {
	peek := 0
	if c.haspeek {
		c.haspeek = false
		b[0] = c.peek[0]
		if len(b) == 1 {
			return 1, nil
		}

		peek = 1
		b = b[peek:]
	}

	n, err := c.Conn.Read(b)
	return n + peek, err
}
