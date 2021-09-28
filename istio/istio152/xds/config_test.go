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

package xds

import (
	"encoding/json"
	"testing"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	xdsbootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/golang/protobuf/jsonpb"
	"mosn.io/mosn/pkg/config/v2"
)

func TestLoadXdsConfig(t *testing.T) {
	cfg := &v2.MOSNConfig{}
	content := []byte(xdsSdsConfig)
	if err := json.Unmarshal(content, cfg); err != nil {
		t.Fatal("json unmarshal config failed, ", xdsSdsConfig, "", err)
	}

	staticResource := &xdsbootstrap.Bootstrap_StaticResources{}
	jContent, err := cfg.RawStaticResources.MarshalJSON()
	if err != nil {
		t.Fatalf("parse pilot address fail: %v", err)
	}

	err = jsonpb.UnmarshalString(string(jContent), staticResource)
	if err != nil {
		t.Fatalf("unmarshal static resource fail %v", err)
	}

	if staticResource.Clusters[0].Name != "xds-grpc" {
		t.Fatal("cluster name should be xds-grpc")
	}

	firstHost := staticResource.Clusters[0].Hosts[0]
	hAddress := firstHost.Address.(*core.Address_SocketAddress)
	if hAddress.SocketAddress.Address != "pilot.test" {
		t.Fatal("parse pilot name fail")
	}

}
