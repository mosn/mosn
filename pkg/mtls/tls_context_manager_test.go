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

	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/mtls/crypto/tls"
	"mosn.io/mosn/pkg/types"
)

func mockGenHashValue(_ *tls.Config) *types.HashValue {
	return nil
}

func TestGetConfigForClient(t *testing.T) {
	config1 := &tls.Config{
		ServerName: "something different",
	}
	provider1 := &staticProvider{
		tlsContext: &tlsContext{
			matches: make(map[string]struct{}),
			server:  types.NewTLSConfigContext(config1, mockGenHashValue),
		},
	}
	config2 := &tls.Config{
		ServerName: "a.com",
	}
	provider2 := &staticProvider{
		tlsContext: &tlsContext{
			matches: map[string]struct{}{"a.com": {}, "http/1.1": {}},
			server:  types.NewTLSConfigContext(config2, mockGenHashValue),
		},
	}
	config3 := &tls.Config{
		ServerName: "b.com",
	}
	provider3 := &staticProvider{
		tlsContext: &tlsContext{
			matches: map[string]struct{}{"b.com": {}, "http/1.1": {}},
			server:  types.NewTLSConfigContext(config3, mockGenHashValue),
		},
	}
	cm := &serverContextManager{
		providers: []types.TLSProvider{provider1, provider2, provider3},
	}
	testCases := []struct {
		name   string
		info   *tls.ClientHelloInfo
		config *tls.Config
	}{
		{
			name: "serverName matched",
			info: &tls.ClientHelloInfo{
				ServerName:      "b.com",
				SupportedProtos: []string{"http/1.1"},
			},
			config: config3,
		},
		{
			name: "no serverName matched but alpn matched",
			info: &tls.ClientHelloInfo{
				ServerName:      "c.com",
				SupportedProtos: []string{"http/1.1"},
			},
			config: config2,
		},
		{
			name: "no matched use default",
			info: &tls.ClientHelloInfo{
				ServerName:      "c.com",
				SupportedProtos: []string{"h2"},
			},
			config: config1,
		},
	}
	for _, testCase := range testCases {
		c, err := cm.GetConfigForClient(testCase.info)
		assert.Nil(t, err)
		assert.Equal(t, testCase.config.ServerName, c.ServerName)
	}
}
