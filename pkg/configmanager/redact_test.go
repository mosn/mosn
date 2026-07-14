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

package configmanager

import (
	"encoding/json"
	"strings"
	"testing"

	v2 "mosn.io/mosn/pkg/config/v2"
)

const secretPrivateKey = "-----BEGIN PRIVATE KEY-----\nSUPER-SECRET\n-----END PRIVATE KEY-----"

// withInlineTLS builds an effectiveConfig where TLS private keys are stored
// inline in every location that config_dump could expose: cluster manager,
// listener filter chains (single and set forms) and clusters.
func withInlineTLS() effectiveConfig {
	src := effectiveConfig{
		Listener: make(map[string]v2.Listener),
		Cluster:  make(map[string]v2.Cluster),
		Routers:  make(map[string]v2.RouterConfiguration),
	}

	// cluster manager tls context
	src.MosnConfig.ClusterManager.TLSContext = v2.TLSConfig{PrivateKey: secretPrivateKey}

	// listener with a single tls_context
	l1 := v2.Listener{}
	l1.Name = "listener-single"
	l1.FilterChains = append(l1.FilterChains, v2.FilterChain{
		TLSContexts: []v2.TLSConfig{{PrivateKey: secretPrivateKey}},
	})
	src.Listener["listener-single"] = l1

	// listener with a tls_context_set (multiple), serialized via TLSConfigs
	l2 := v2.Listener{}
	l2.Name = "listener-set"
	l2.FilterChains = append(l2.FilterChains, v2.FilterChain{
		TLSContexts: []v2.TLSConfig{
			{PrivateKey: secretPrivateKey},
			{PrivateKey: secretPrivateKey},
		},
	})
	src.Listener["listener-set"] = l2

	// cluster tls context
	src.Cluster["cluster-tls"] = v2.Cluster{Name: "cluster-tls", TLS: v2.TLSConfig{PrivateKey: secretPrivateKey}}

	return src
}

func TestRedactedCopyRedactsAllInlinePrivateKeys(t *testing.T) {
	src := withInlineTLS()
	dst := redactedCopy(src)

	// the in-memory redacted snapshot must not carry the raw private key
	if check := func(s string) {
		if strings.Contains(s, secretPrivateKey) {
			t.Fatalf("redacted snapshot still contains raw private key")
		}
	}; true {
		b, err := json.Marshal(dst)
		if err != nil {
			t.Fatal(err)
		}
		check(string(b))
	}

	// and the original source must remain untouched (redaction is on the copy)
	if src.MosnConfig.ClusterManager.TLSContext.PrivateKey != secretPrivateKey {
		t.Fatalf("redaction mutated the source config")
	}
	if src.Listener["listener-single"].FilterChains[0].TLSContexts[0].PrivateKey != secretPrivateKey {
		t.Fatalf("redaction mutated the source listener config")
	}
	if src.Cluster["cluster-tls"].TLS.PrivateKey != secretPrivateKey {
		t.Fatalf("redaction mutated the source cluster config")
	}
}

func TestDumpJSONRedactsPrivateKeys(t *testing.T) {
	Reset()
	defer Reset()

	// seed the global effective config with inline TLS material
	src := withInlineTLS()
	conf = src

	buf, err := DumpJSON()
	if err != nil {
		t.Fatal(err)
	}
	out := string(buf)
	if strings.Contains(out, secretPrivateKey) {
		t.Fatalf("DumpJSON leaked the private key: %s", out)
	}
	if !strings.Contains(out, redactedPrivateKey) {
		t.Fatalf("DumpJSON did not emit the redaction placeholder")
	}
}

func TestHandleMOSNConfigRedactsPrivateKeys(t *testing.T) {
	Reset()
	defer Reset()

	src := withInlineTLS()
	conf = src

	cases := []string{CfgTypeMOSN, CfgTypeListener, CfgTypeCluster}
	for _, typ := range cases {
		var got string
		HandleMOSNConfig(typ, func(v interface{}) {
			b, err := json.MarshalIndent(v, "", " ")
			if err != nil {
				t.Fatal(err)
			}
			got = string(b)
		})
		if strings.Contains(got, secretPrivateKey) {
			t.Fatalf("HandleMOSNConfig(%s) leaked the private key", typ)
		}
	}
}
