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
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

// staticProvider is an implementation of types.Provider
// staticProvider stored a static certificate
type staticProvider struct {
	*tlsContext
}

func (p *staticProvider) Ready() bool {
	return true
}

func (p *staticProvider) Empty() bool {
	return p.tlsContext.server == nil
}

// NewProvider returns a types.Provider.
// we support sds provider and static provider.
func NewProvider(index string, cfg *v2.TLSConfig) (types.TLSProvider, error) {
	if !cfg.Status {
		return nil, nil
	}
	if cfg.SdsConfig != nil {
		if !cfg.SdsConfig.Valid() {
			return nil, ErrorNoCertConfigure
		}
		return addOrUpdateProvider(index, cfg), nil
	} else {
		// static provider
		secret := &SecretInfo{
			Certificate: cfg.CertChain,
			PrivateKey:  cfg.PrivateKey,
			Validation:  cfg.CACert,
		}
		ctx, err := newTLSContext(cfg, secret)
		if err != nil {
			return nil, err
		}
		return &staticProvider{
			tlsContext: ctx,
		}, nil
	}
}
