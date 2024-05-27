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
	"testing"

	v2 "mosn.io/mosn/pkg/config/v2"
)

// Test Yaml Config Load
func TestYamlConfigLoad(t *testing.T) {
	dynamic_mode := Load("testdata/envoy.yaml")
	if !(len(dynamic_mode.RawStaticResources) > 0 &&
		len(dynamic_mode.RawDynamicResources) > 0) {
		t.Fatalf("load dynamic yaml config failed")
	}
	static_mode := Load("testdata/config.yaml")
	if !(static_mode.Servers[0].Listeners[0].Addr != nil &&
		len(static_mode.ClusterManager.Clusters) == 1 &&
		static_mode.ClusterManager.Clusters[0].Name == "example") {
		t.Fatalf("load static yaml config failed")
	}
}

// Test Register Load Config Func
func TestRegisterConfigLoadFunc(t *testing.T) {
	RegisterConfigLoadFunc(func(p string) *v2.MOSNConfig {
		return &v2.MOSNConfig{
			Pid: "test",
		}
	})
	defer func() {
		RegisterConfigLoadFunc(DefaultConfigLoad)
	}()
	cfg := Load("test")
	if cfg.Pid != "test" {
		t.Fatalf("register config load func failed")
	}
}
