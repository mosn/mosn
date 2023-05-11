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
	"context"
	"errors"
	"fmt"
	"sync"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls/certtool"
	"mosn.io/mosn/pkg/types"
)

type mockSdsClient struct {
	lock     sync.Mutex
	callback map[string]types.SdsUpdateCallbackFunc
}

var mockSdsClientInstance *mockSdsClient

func (c *mockSdsClient) AddUpdateCallback(name string, f types.SdsUpdateCallbackFunc) error {
	c.lock.Lock()
	c.callback[name] = f
	c.lock.Unlock()
	return nil
}

func (c *mockSdsClient) SetSecret(name string, secret *types.SdsSecret) {
	c.lock.Lock()
	defer c.lock.Unlock()
	f, ok := c.callback[name]
	if !ok {
		log.DefaultLogger.Errorf("[unit test] name %s with no callback", name)
		return
	}
	f(name, secret)
}

func (c *mockSdsClient) AckResponse(resp interface{}) {
}

func (c *mockSdsClient) DeleteUpdateCallback(name string) error {
	return nil
}

func (c *mockSdsClient) RequireSecret(name string) {
}

func (c *mockSdsClient) FetchSecret(ctx context.Context, name string) (*types.SdsSecret, error) {
	return nil, errors.New("not implement yet")
}

func getMockSdsClient(cfg interface{}) types.SdsClient {
	if mockSdsClientInstance == nil {
		mockSdsClientInstance = &mockSdsClient{
			callback: make(map[string]types.SdsUpdateCallbackFunc),
		}
	}
	return mockSdsClientInstance
}

type certInfo struct {
	CommonName string
	Curve      string
	DNS        string
}

func (c *certInfo) CreateSecret() (*SecretInfo, error) {
	priv, err := certtool.GeneratePrivateKey(c.Curve)
	if err != nil {
		return nil, fmt.Errorf("generate key failed %v", err)
	}
	var dns []string
	if c.DNS != "" {
		dns = append(dns, c.DNS)
	}
	tmpl, err := certtool.CreateTemplate(c.CommonName, false, dns)
	if err != nil {
		return nil, fmt.Errorf("generate certificate template failed %v", err)
	}
	cert, err := certtool.SignCertificate(tmpl, priv)
	if err != nil {
		return nil, fmt.Errorf("sign certificate failed %v", err)
	}
	return &SecretInfo{
		Certificate: cert.CertPem,
		PrivateKey:  cert.KeyPem,
		Validation:  certtool.GetRootCA().CertPem,
	}, nil
}

func (c *certInfo) CreateCertConfig() (*v2.TLSConfig, error) {
	secret, err := c.CreateSecret()
	if err != nil {
		return nil, err
	}
	return &v2.TLSConfig{
		Status:     true,
		CACert:     secret.Validation,
		CertChain:  secret.Certificate,
		PrivateKey: secret.PrivateKey,
	}, nil
}
