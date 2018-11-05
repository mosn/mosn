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
	"fmt"
	"net"
	"strings"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/mtls/crypto/tls"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type context struct {
	matches     map[string]bool
	listener    types.Listener
	clusterInfo types.ClusterInfo
	tlsConfig   *tls.Config
	serverName  string
	ticket      string
}

func (ctx *context) buildMatch() {
	if ctx.tlsConfig == nil {
		return
	}
	match := make(map[string]bool)
	certs := ctx.tlsConfig.Certificates
	for i := range certs {
		cert := certs[i]
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			continue
		}
		if len(x509Cert.Subject.CommonName) > 0 {
			match[x509Cert.Subject.CommonName] = true
		}
		for _, san := range x509Cert.DNSNames {
			match[san] = true
		}
	}
	for _, protocol := range ctx.tlsConfig.NextProtos {
		match[protocol] = true
	}
	match[ctx.serverName] = true
	ctx.matches = match
}

type contextManager struct {
	contexts  []*context
	logger    log.Logger
	isClient  bool
	inspector bool
	listener  types.Listener
	server    *tls.Config
	//	mutex sync.RWMutex
}

// NewTLSServerContextManager returns a types.TLSContextManager used in TLS Server
// A Server Manager can contains multiple certificate context
// The server is a tls.Config that used to find real tls.Config in contexts, just valid in server
func NewTLSServerContextManager(config *v2.Listener, l types.Listener, logger log.Logger) (types.TLSContextManager, error) {
	mgr := &contextManager{
		contexts:  []*context{},
		logger:    logger,
		listener:  l,
		inspector: config.Inspector,
		//mutex:    sync.RWMutex{},
	}
	mgr.server = &tls.Config{
		GetConfigForClient: mgr.GetConfigForClient,
	}
	for _, c := range config.FilterChains {
		if err := mgr.AddContext(&c.TLS); err != nil {
			return nil, err
		}
	}

	return mgr, nil
}

// NewTLSClientContextManager returns a types.TLSContextManager used in TLS Client
// Client Manager just have one context
func NewTLSClientContextManager(config *v2.TLSConfig, info types.ClusterInfo) (types.TLSContextManager, error) {
	mgr := &contextManager{
		logger:   log.DefaultLogger,
		isClient: true,
	}
	if err := mgr.AddContext(config); err != nil {
		return nil, err
	}
	if len(mgr.contexts) != 0 {
		mgr.contexts[0].clusterInfo = info
	}
	return mgr, nil
}

func (mgr *contextManager) newTLSConfig(c *v2.TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	factory := getFactory(c.Type)
	hooks := factory.CreateConfigHooks(c.ExtendVerify)
	if c.CipherSuites != "" {
		ciphers := strings.Split(c.CipherSuites, ":")
		for _, s := range ciphers {
			cipher, ok := ciphersMap[s]
			if !ok {
				return nil, fmt.Errorf("cipher %s is not supported", s)
			}
			tlsConfig.CipherSuites = append(tlsConfig.CipherSuites, cipher)
		}
		if len(tlsConfig.CipherSuites) == 0 {
			tlsConfig.CipherSuites = defaultCiphers
		}
	}
	if c.EcdhCurves != "" {
		curves := strings.Split(c.EcdhCurves, ",")
		for _, s := range curves {
			curve, ok := allCurves[strings.ToLower(s)]
			if !ok {
				return nil, fmt.Errorf("curve %s is not supported", s)
			}
			tlsConfig.CurvePreferences = append(tlsConfig.CurvePreferences, curve)
		}
		if len(tlsConfig.CurvePreferences) == 0 {
			tlsConfig.CurvePreferences = defaultCurves
		}
	}
	tlsConfig.MaxVersion = maxProtocols
	if c.MaxVersion != "" {
		protocol, ok := version[strings.ToLower(c.MaxVersion)]
		if !ok {
			return nil, fmt.Errorf("tls protocol %s is not supported", c.MaxVersion)
		}
		if protocol != 0 {
			tlsConfig.MaxVersion = protocol
		}
	}
	tlsConfig.MinVersion = minProtocols
	if c.MinVersion != "" {
		protocol, ok := version[strings.ToLower(c.MinVersion)]
		if !ok {
			return nil, fmt.Errorf("tls protocol %s is not supported", c.MaxVersion)
		}
		if protocol != 0 {
			tlsConfig.MinVersion = protocol
		}
	}
	if c.ALPN != "" {
		protocols := strings.Split(c.ALPN, ",")
		for _, p := range protocols {
			_, ok := alpn[strings.ToLower(p)]
			if !ok {
				return nil, fmt.Errorf("ALPN %s is not supported", p)
			}
			tlsConfig.NextProtos = append(tlsConfig.NextProtos, p)
		}
	}
	cert, err := hooks.GetCertificate(c.CertChain, c.PrivateKey)
	switch err {
	case ErrorNoCertConfigure: // cert/key config is empty string, no certificate
		if !mgr.isClient {
			return nil, ErrorGetCertificateFailed
		}
	case nil:
		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
	default: //other error
		return nil, ErrorGetCertificateFailed
	}
	// pool can be nil, if it is nil, TLS uses the host's root CA set.
	pool, err := hooks.GetX509Pool(c.CACert)
	if err != nil {
		return nil, err
	}
	// VerifyClient is valid when isClient is false
	// InsecureSkip is valid when isClient is true
	if mgr.isClient {
		tlsConfig.ServerName = c.ServerName
		tlsConfig.RootCAs = pool
		verify := hooks.VerifyPeerCertificate()
		if verify != nil {
			// use self verify, skip normal verify
			tlsConfig.InsecureSkipVerify = true
			tlsConfig.VerifyPeerCertificate = verify
		}
		if c.InsecureSkip {
			tlsConfig.InsecureSkipVerify = true
			tlsConfig.VerifyPeerCertificate = nil
		}
	} else { //Server
		if c.VerifyClient {
			tlsConfig.ClientCAs = pool
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
			tlsConfig.VerifyPeerCertificate = hooks.VerifyPeerCertificate()
		}
	}
	return tlsConfig, nil
}

func (mgr *contextManager) AddContext(c *v2.TLSConfig) error {
	if !c.Status {
		return nil
	}
	if mgr.isClient && len(mgr.contexts) >= 1 {
		return errors.New("client manager support only one context")
	}
	tlsConfig, err := mgr.newTLSConfig(c)
	if err != nil {
		if c.Fallback && err == ErrorGetCertificateFailed {
			mgr.logger.Warnf("something wrong with certificate/key, trigger fallback")
			return nil
		}
		return err
	}
	ctx := &context{
		listener:   mgr.listener,
		serverName: c.ServerName,
		tlsConfig:  tlsConfig,
		ticket:     c.Ticket,
	}
	ctx.buildMatch()
	mgr.contexts = append(mgr.contexts, ctx)
	return nil
}

func (mgr *contextManager) GetConfigForClient(info *tls.ClientHelloInfo) (*tls.Config, error) {
	if !mgr.Enabled() {
		return nil, errors.New("no certificate context in context manager")
	}
	var tlscontext *context
	// match context in order
	for _, ctx := range mgr.contexts {
		// first match ServerName
		// e.g. www.example.com will be first matched against www.example.com, then *.example.com, then *.com
		if info.ServerName != "" {
			name := strings.ToLower(info.ServerName)
			for len(name) > 0 && name[len(name)-1] == '.' {
				name = name[:len(name)-1]
			}
			if _, ok := ctx.matches[name]; ok {
				tlscontext = ctx
				goto find
			}
			labels := strings.Split(name, ".")
			for i := 0; i < len(labels)-1; i++ {
				labels[i] = "*"
				candidate := strings.Join(labels[i:], ".")
				if _, ok := ctx.matches[candidate]; ok {
					tlscontext = ctx
					goto find
				}
			}
		}
		// Sencond match ALPN
		for _, protocol := range info.SupportedProtos {
			protocol = strings.ToLower(protocol)
			if _, ok := ctx.matches[protocol]; ok {
				tlscontext = ctx
				goto find
			}
		}
	}
	// Last, return the first certificate.
	tlscontext = mgr.contexts[0]
find:
	// TODO:
	// callback select filter config
	// callback(cm.listener, index)
	return tlscontext.tlsConfig.Clone(), nil
}

func (mgr *contextManager) Enabled() bool {
	return len(mgr.contexts) != 0
}
func (mgr *contextManager) Config() *tls.Config {
	if !mgr.Enabled() {
		return nil
	}
	if mgr.isClient {
		return mgr.contexts[0].tlsConfig.Clone()
	}
	return mgr.server.Clone()
}
func (mgr *contextManager) Conn(c net.Conn) net.Conn {
	if _, ok := c.(*net.TCPConn); !ok {
		return c
	}
	if !mgr.Enabled() {
		return c
	}
	if mgr.isClient {
		return getTLSConn(c, mgr.Config(), mgr.isClient)
	}
	if !mgr.inspector {
		return getTLSConn(c, mgr.Config(), mgr.isClient)
	}
	// do inspector
	conn := &Conn{
		Conn: c,
	}
	buf := conn.Peek()
	if buf == nil {
		return getTLSConn(conn, mgr.Config(), mgr.isClient)
	}
	switch buf[0] {
	// TLS handshake
	case 0x16:
		return getTLSConn(conn, mgr.Config(), mgr.isClient)
	// Non TLS
	default:
		return conn
	}
}

func getTLSConn(c net.Conn, config *tls.Config, isClient bool) net.Conn {
	if isClient {
		return &TLSConn{
			tls.Client(c, config),
		}
	}

	return &TLSConn{
		tls.Server(c, config),
	}
}
