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

package server

import (
	"errors"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
)

func initStoreConfig() {
	configmanager.Reset()
	configmanager.SetMosnConfig(&v2.MOSNConfig{
		Tracing: v2.TracingConfig{
			Enable: true,
		},
	})
	configmanager.SetRouter(v2.RouterConfiguration{
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: "test_router",
		},
	})
	configmanager.SetClusterConfig(v2.Cluster{
		Name: "test_cluster",
		Hosts: []v2.Host{
			{
				HostConfig: v2.HostConfig{
					Address: "127.0.0.1:8080",
				},
			},
		},
	})
	configmanager.SetListenerConfig(v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:       "test_listener",
			AddrConfig: "127.0.0.1:8080",
		},
	})
}

func TestDumpConfigWithParameters(t *testing.T) {
	initStoreConfig()
	testCases := []struct {
		url    string
		verify func(b []byte) error
	}{
		{
			url: "http://127.0.0.1/api/v1/config_dump?allrouters",
			verify: func(b []byte) error {
				result := map[string]v2.RouterConfigurationConfig{}
				if err := json.Unmarshal(b, &result); err != nil {
					return err
				}
				if len(result) != 1 {
					return errors.New("result is not expected")
				}
				return nil
			},
		},
		{
			url: "http://127.0.0.1/api/v1/config_dump?router=test_router",
			verify: func(b []byte) error {
				result := v2.RouterConfigurationConfig{}
				if err := json.Unmarshal(b, &result); err != nil {
					return err
				}
				if result.RouterConfigName != "test_router" {
					return errors.New("result is not expected")
				}
				return nil
			},
		},
		{
			url: "http://127.0.0.1/api/v1/config_dump?allclusters",
			verify: func(b []byte) error {
				result := map[string]v2.Cluster{}
				if err := json.Unmarshal(b, &result); err != nil {
					return err
				}
				if len(result) != 1 {
					return errors.New("result is not expected")
				}
				return nil
			},
		},
		{
			url: "http://127.0.0.1/api/v1/config_dump?cluster=test_cluster",
			verify: func(b []byte) error {
				result := v2.Cluster{}
				if err := json.Unmarshal(b, &result); err != nil {
					return err
				}
				if result.Name != "test_cluster" && len(result.Hosts) != 1 {
					return errors.New("result is not expected")
				}
				return nil
			},
		},
		{
			url: "http://127.0.0.1/api/v1/config_dump?alllisteners",
			verify: func(b []byte) error {
				result := map[string]v2.Listener{}
				if err := json.Unmarshal(b, &result); err != nil {
					return err
				}
				if len(result) != 1 {
					return errors.New("result is not expected")
				}
				return nil
			},
		},
		{
			url: "http://127.0.0.1/api/v1/config_dump?listener=test_listener",
			verify: func(b []byte) error {
				result := v2.Listener{}
				if err := json.Unmarshal(b, &result); err != nil {
					return err
				}
				if result.AddrConfig != "127.0.0.1:8080" {
					return errors.New("result is not expected")
				}
				return nil
			},
		},
		{
			url: "http://127.0.0.1/api/v1/config_dump?mosnconfig",
			verify: func(b []byte) error {
				result := v2.MOSNConfig{}
				if err := json.Unmarshal(b, &result); err != nil {
					return err
				}
				if !result.Tracing.Enable {
					return errors.New("result is not expected")
				}
				return nil
			},
		},
	}
	for _, tc := range testCases {
		r := httptest.NewRequest("GET", tc.url, nil)
		w := httptest.NewRecorder()
		ConfigDump(w, r)
		resp := w.Result()
		if resp.StatusCode != 200 {
			t.Fatalf("response is not 200, got:%d", resp.StatusCode)
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read response error: %v", err)
		}
		if err := tc.verify(b); err != nil {
			t.Fatalf("verify result error: %v, body: %s", err, string(b))
		}
	}
}
