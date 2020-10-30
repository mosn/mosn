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
	"os"
	"testing"

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/pkg/utils"
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

func TestLoadDynamicConfig(t *testing.T) {
	config1 := `
	{
	  "servers": [
	    {
	      "listeners": [
	        {
	          "name": "testListener",
	          "address": "0.0.0.0:28089",
	          "bind_port": true,
	          "inspector": true,
	          "filter_chains": [
	            {
	              "filters": [
	                {
	                  "config": {
	                    "downstream_protocol": "Auto",
	                    "router_config_name": "http.19002",
	                    "upstream_protocol": "Auto"
	                  },
	                  "type": "proxy"
	                }
	              ]
	            }
	          ]
	        }
	      ]
	    }
	  ]
	}
	`
	config2 := `
    {
      "servers": [
        {
          "listeners": [
            {
              "name": "testListener1",
              "address": "0.0.0.0:19002",
              "bind_port": true,
              "inspector": true,
              "filter_chains": [
                {
                  "filters": [
                    {
                      "config": {
                        "downstream_protocol": "Auto",
                        "router_config_name": "http.28089",
                        "upstream_protocol": "Auto"
                      },
                      "type": "proxy"
                    }
                  ]
                }
              ]
            },
            {
              "name": "testListener",
              "address": "0.0.0.0:28089",
              "bind_port": true,
              "inspector": true,
              "filter_chains": [
                {
                  "filters": [
                    {
                      "config": {
                        "downstream_protocol": "Auto",
                        "router_config_name": "http.28089",
                        "upstream_protocol": "Auto"
                      },
                      "type": "proxy"
                    }
                  ]
                }
              ]
            }
          ],
          "routers": [
            {
              "router_config_name": "http.19002",
              "virtual_hosts": [
                {
                  "name": "*:19002",
                  "domains": [
                    "*"
                  ],
                  "routers": [
                    {
                      "match": {
                        "prefix": "/"
                      },
                      "route": {
                        "cluster_name": "testCluster"
                      }
                    }
                  ]
                }
              ]
            }
          ]
        }
      ],
	 "cluster_manager": {
	     "clusters": [
	       {
	         "name": "cmagentCluster",
	         "type": "STRICT_DNS",
	         "lb_type": "LB_ROUNDROBIN",
	         "max_request_per_conn": 1024,
	         "conn_buffer_limit_bytes": 32768,
	         "circuit_breakers": null,
	         "health_check": {
	           "timeout": "0s",
	           "interval": "0s",
	           "interval_jitter": "0s"
	         }
	       }
	     ]
	   }
   }`
	f1 := "/tmp/config1.json"
	f2 := "/tmp/config2.json"
	utils.WriteFileSafety(f1, []byte(config1), os.ModePerm)
	utils.WriteFileSafety(f2, []byte(config2), os.ModePerm)
	SetDynamicConfigPath(f2)
	conf := Load(f1)
	if conf == nil {
		t.Fatalf("load config and update from dyconfig file failed")
	}
	if len(conf.Servers) != 1 {
		t.Fatalf("load config and update from dyconfig file failed")
	}
	if len(conf.Servers[0].Routers) != 1 {
		t.Fatalf("load config and update router from dyconfig file failed")
	}
	if len(conf.Servers[0].Listeners) != 2 {
		t.Fatalf("load config and update listener from dyconfig file failed")
	}
	if len(conf.ClusterManager.Clusters) != 1 {
		t.Fatalf("load config and update cluster from dyconfig file failed")
	}
}
