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
	"net"
	"reflect"

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls/crypto/tls"
	"mosn.io/mosn/pkg/types"
)

type serverContextManager struct {
	// providers stored the certificates
	providers []types.TLSProvider
	// if inspector is true, the manager support non-tls connections
	inspector bool
	// config is a tls.config with GetConfigForClient
	config *tls.Config
}

// NewTLSServerContextManager returns a types.TLSContextManager used in TLS Server
// A Server Manager can contains multiple certificates in provider
func NewTLSServerContextManager(cfg *v2.Listener) (types.TLSContextManager, error) {
	mng := &serverContextManager{
		inspector: cfg.Inspector,
	}
	mng.config = &tls.Config{
		GetConfigForClient: mng.GetConfigForClient,
	}
	for _, c := range cfg.FilterChains {
		for _, tlsCfg := range c.TLSContexts {
			provider, err := NewProvider(&tlsCfg)
			if err != nil {
				return nil, err
			}
			// provider is an interface, needs to check by reflect
			if provider != nil && !reflect.ValueOf(provider).IsNil() {
				// if a server receive a empty provider and do not support fallback, it should be failed
				if provider.Empty() {
					if !tlsCfg.Fallback {
						return nil, ErrorNoCertConfigure
					}
					log.DefaultLogger.Alertf(types.ErrorKeyTLSFallback, "listener enable tls without certificate, fallback tls")
				} else {
					mng.providers = append(mng.providers, provider)
				}
			}
		}
	}
	return mng, nil
}

func (mng *serverContextManager) GetConfigForClient(info *tls.ClientHelloInfo) (*tls.Config, error) {
	var defaultProvider types.TLSProvider
	for _, provider := range mng.providers {
		if !provider.Ready() {
			continue
		}
		// the default provider is the first provider which is ready
		if defaultProvider == nil {
			defaultProvider = provider
		}
		if provider.MatchedServerName(info.ServerName) {
			return provider.GetTLSConfig(false), nil
		}
		if provider.MatchedALPN(info.SupportedProtos) {
			return provider.GetTLSConfig(false), nil
		}
	}
	if defaultProvider == nil {
		return nil, ErrorNoCertConfigure
	}
	return defaultProvider.GetTLSConfig(false), nil
}

func (mng *serverContextManager) Conn(c net.Conn) (net.Conn, error) {
	if _, ok := c.(*net.TCPConn); !ok {
		return c, nil
	}
	if !mng.Enabled() {
		return c, nil
	}
	if !mng.inspector {
		return &TLSConn{
			tls.Server(c, mng.config.Clone()),
		}, nil
	}
	// inspector
	conn := &Conn{
		Conn: c,
	}
	buf, err := conn.Peek()
	if err != nil {
		return nil, err
	}
	switch buf[0] {
	// TLS handshake
	case 0x16:
		return &TLSConn{
			tls.Server(conn, mng.config.Clone()),
		}, nil
	// Non TLS
	default:
		return conn, nil
	}
}

func (mng *serverContextManager) Enabled() bool {
	for _, p := range mng.providers {
		if p.Ready() {
			return true
		}
	}
	return false
}

type clientContextManager struct {
	// client support only one certificate
	provider types.TLSProvider
}

// NewTLSClientContextManager returns a types.TLSContextManager used in TLS Client
func NewTLSClientContextManager(cfg *v2.TLSConfig) (types.TLSContextManager, error) {
	provider, err := NewProvider(cfg)
	if err != nil {
		return nil, err
	}
	mng := &clientContextManager{
		provider: provider,
	}
	return mng, nil
}

func (mng *clientContextManager) Conn(c net.Conn) (net.Conn, error) {
	if _, ok := c.(*net.TCPConn); !ok {
		return c, nil
	}
	if !mng.Enabled() {
		return c, nil
	}
	return &TLSConn{
		tls.Client(c, mng.provider.GetTLSConfig(true)),
	}, nil
}

func (mng *clientContextManager) Enabled() bool {
	return mng.provider != nil && mng.provider.Ready()
}
