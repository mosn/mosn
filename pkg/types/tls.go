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

package types

import (
	"net"

	"sofastack.io/sofa-mosn/pkg/mtls/crypto/tls"
)

// TLSContextManager manages the listener/cluster's tls config
type TLSContextManager interface {
	// Conn handles the connection, makes a connection as tls connection
	// or keep it as a non-tls connection
	Conn(net.Conn) net.Conn
}

// TLSProvider provides a tls config for connection
// the matched function is used for check whether the connection should use this provider
type TLSProvider interface {
	// GetTLSConfig returns the tls config used in connection
	// if client is true, return the client mode config, or returns the server mode config
	GetTLSConfig(client bool) *tls.Config
	// MatchedServerName checks whether the server name is matched the stored tls certificate
	MatchedServerName(sn string) bool
	// MatchedALPN checks whether the ALPN is matched the stored tls certificate
	MatchedALPN(protos []string) bool
	// Ready checks whether the provider is inited.
	// the static provider should be always ready.
	Ready() bool
	// Empty represent whether the provider contains a certificate or not.
	// A Ready Provider maybe empty too.
	// the sds provider should be always not empty.
	Empty() bool
}

type SDSSecret struct {
	Name           string
	CertificatePEM string
	PrivateKeyPEM  string
	ValidationPEM  string
}
