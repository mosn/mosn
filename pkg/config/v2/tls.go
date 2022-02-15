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

package v2

import (
	"bytes"
	"encoding/json"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

// TLSConfig is a configuration of tls context
type TLSConfig struct {
	Status            bool                   `json:"status,omitempty"`
	Type              string                 `json:"type,omitempty"`
	ServerName        string                 `json:"server_name,omitempty"`
	CACert            string                 `json:"ca_cert,omitempty"`
	CertChain         string                 `json:"cert_chain,omitempty"`
	PrivateKey        string                 `json:"private_key,omitempty"`
	VerifyClient      bool                   `json:"verify_client,omitempty"`
	RequireClientCert bool                   `json:"require_client_cert,omitempty"`
	InsecureSkip      bool                   `json:"insecure_skip,omitempty"`
	CipherSuites      string                 `json:"cipher_suites,omitempty"`
	EcdhCurves        string                 `json:"ecdh_curves,omitempty"`
	MinVersion        string                 `json:"min_version,omitempty"`
	MaxVersion        string                 `json:"max_version,omitempty"`
	ALPN              string                 `json:"alpn,omitempty"`
	Ticket            string                 `json:"ticket,omitempty"`
	Fallback          bool                   `json:"fall_back,omitempty"`
	ExtendVerify      map[string]interface{} `json:"extend_verify,omitempty"`
	Callbacks         []string               `json:"callbacks,omitempty"`
	SdsConfig         *SdsConfig             `json:"sds_source,omitempty"`
}

type SdsConfig struct {
	CertificateConfig *SecretConfigWrapper
	ValidationConfig  *SecretConfigWrapper
}

type SecretConfigWrapper struct {
	Name      string      `json:"-"`
	SdsConfig interface{} `json:"-"`
	raw       SecretConfigWrapperConfig
}

func (scw SecretConfigWrapper) MarshalJSON() (b []byte, err error) {
	// if we build SecretConfigWrapper directly, we should use a proto.Message to build it.
	if pm, ok := scw.SdsConfig.(proto.Message); ok {
		var buf bytes.Buffer
		marshaler := &jsonpb.Marshaler{}
		_ = marshaler.Marshal(&buf, pm)
		// use jsonpb format to marshal
		m := map[string]interface{}{}
		json.Unmarshal(buf.Bytes(), &m)
		scw.raw.SdsConfig = m
	}
	scw.raw.Name = scw.Name
	return json.Marshal(scw.raw)
}

func (scw *SecretConfigWrapper) UnmarshalJSON(b []byte) error {
	cfg := SecretConfigWrapperConfig{}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return err
	}
	scw.Name = cfg.Name
	scw.SdsConfig = cfg.SdsConfig
	scw.raw = cfg
	return nil
}

// Valid checks the whether the SDS Config is valid or not
func (c *SdsConfig) Valid() bool {
	return c != nil && c.CertificateConfig != nil // && c.ValidationConfig != nil // ValidationConfig can be setted to nil
}

type SecretConfigWrapperConfig struct {
	Name      string                 `json:"name"`
	SdsConfig map[string]interface{} `json:"sdsConfig"`
}
