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
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

func TestInheritConfig(t *testing.T) {
	tests := []struct {
		name           string
		testConfigPath string
		mosnConfig     string
		inheritFunc    func(config *v2.MOSNConfig) error
		configData     func(configPath, configStr string) []byte
		wantErr        bool
	}{
		{
			name:           "test Inherit Config",
			testConfigPath: "/tmp/mosn/mosn_admin.json",
			mosnConfig:     mosnConfig,
			inheritFunc: func(c *v2.MOSNConfig) error {
				// inherit old mosn config
				configData, err := GetInheritConfig()
				if err != nil {
					return nil
				}

				oldMosnConfig := &v2.MOSNConfig{}
				err = json.Unmarshal(configData, oldMosnConfig)
				if err != nil {
					return err
				}

				log.StartLogger.Debugf("[mosn] [NewMosn] old mosn config: %v", oldMosnConfig)
				c.Servers = oldMosnConfig.Servers
				c.InheritOldMosnconfig = oldMosnConfig.InheritOldMosnconfig
				c.ClusterManager = oldMosnConfig.ClusterManager
				c.RawAdmin = oldMosnConfig.RawAdmin

				return nil
			},
			configData: func(configPath, configStr string) []byte {
				configmanager.Reset()
				createMosnConfig(configPath, configStr)
				if cfg := configmanager.Load(configPath); cfg != nil {
					configmanager.SetMosnConfig(cfg)
					ln := cfg.Servers[0].Listeners[0]
					configmanager.SetListenerConfig(ln)
					cluster := cfg.ClusterManager.Clusters[0]
					configmanager.SetClusterConfig(cluster)
					router := cfg.Servers[0].Routers[0]
					configmanager.SetRouter(*router)
				}
				types.InitDefaultPath(configmanager.GetConfigPath(), "")
				dumpConfigBytes, err := configmanager.InheritMosnconfig()
				if err != nil {
					t.Errorf("Dump config error: %v", err)
				}
				return dumpConfigBytes
			},
			wantErr: false,
		},
		{
			name:           "test Inherit Config with inherit func",
			testConfigPath: "/tmp/mosn/mosn_admin_1.json",
			mosnConfig:     mosnConfig,
			inheritFunc: func(c *v2.MOSNConfig) error {
				// inherit old mosn config
				configData, err := GetInheritConfig()
				if err != nil {
					return err
				}
				oldMosnConfig := &v2.MOSNConfig{}
				err = json.Unmarshal(configData, oldMosnConfig)
				if err != nil {
					return err
				}
				c.Servers = oldMosnConfig.Servers
				c.InheritOldMosnconfig = oldMosnConfig.InheritOldMosnconfig
				c.RawAdmin = oldMosnConfig.RawAdmin
				return nil
			},
			configData: func(configPath, configStr string) []byte {
				configmanager.Reset()
				createMosnConfig(configPath, configStr)
				if cfg := configmanager.Load(configPath); cfg != nil {
					configmanager.SetMosnConfig(cfg)
					ln := cfg.Servers[0].Listeners[0]
					configmanager.SetListenerConfig(ln)
					router := cfg.Servers[0].Routers[0]
					configmanager.SetRouter(*router)
				}
				types.InitDefaultPath(configmanager.GetConfigPath(), "")
				dumpConfigBytes, err := configmanager.InheritMosnconfig()
				if err != nil {
					t.Errorf("Dump config error: %v", err)
				}
				return dumpConfigBytes
			},

			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dumpConfigBytes := tt.configData(tt.testConfigPath, tt.mosnConfig)
			mosnConfigBytes := make([]byte, 0)
			wg := &sync.WaitGroup{}
			wg.Add(2)

			go func() {
				defer wg.Done()
				if err := SendInheritConfig(); (err != nil) != tt.wantErr {
					t.Errorf("SendInheritConfig() error = %v, wantErr %v", err, tt.wantErr)
				}
			}()

			go func() {
				defer wg.Done()
				effectiveConfig := &v2.MOSNConfig{}
				err := tt.inheritFunc(effectiveConfig)
				if err != nil {
					t.Errorf("json marshal effective Config error: %v", err)
				}
				mosnConfigBytes, err = json.MarshalIndent(effectiveConfig, "", "  ")
				if err != nil {
					t.Errorf("json marshal effective Config error: %v", err)
				}
			}()
			wg.Wait()

			if string(dumpConfigBytes) != string(mosnConfigBytes) {
				t.Errorf("error server.GetInheritConfig:, want: %v, but: %v", string(dumpConfigBytes), string(mosnConfigBytes))
			}
		})
	}
}

func createMosnConfig(testConfigPath, config string) {

	os.Remove(testConfigPath)
	os.MkdirAll(filepath.Dir(testConfigPath), 0755)

	ioutil.WriteFile(testConfigPath, []byte(config), 0644)

}

const mosnConfig = `{
  "servers": [
    {
      "default_log_path": "stdout",
      "default_log_level": "DEBUG",
      "graceful_timeout": "0s",
      "listeners": [
        {
          "name": "serverListener",
          "address": "127.0.0.1:2046",
          "bind_port": true,
          "network": "tcp",
          "filter_chains": [
            {
              "tls_context_set": [
                {}
              ],
              "filters": [
                {
                  "type": "proxy",
                  "config": {
                    "downstream_protocol": "Http1",
                    "router_config_name": "server_router",
                    "upstream_protocol": "Http1"
                  }
                }
              ]
            }
          ]
        }
      ],
      "routers": [
        {
          "router_config_name": "server_router",
          "virtual_hosts": [
            {
              "name": "serverHost",
              "domains": [
                "*"
              ],
              "routers": [
                {
                  "match": {
                    "prefix": "/"
                  },
                  "route": {
                    "cluster_name": "serverCluster",
                    "timeout": "0s"
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
    "tls_context": {},
    "clusters": [
      {
        "name": "serverCluster",
        "type": "SIMPLE",
        "lb_type": "LB_RANDOM",
        "max_request_per_conn": 1024,
        "conn_buffer_limit_bytes": 32768,
        "circuit_breakers": null,
        "health_check": {
          "timeout": "0s",
          "interval": "0s",
          "interval_jitter": "0s"
        },
        "spec": {},
        "lb_subset_config": {},
        "original_dst_lb_config": {},
        "tls_context": {},
        "hosts": [
          {
            "address": "127.0.0.1:8080",
            "weight": 1
          }
        ],
        "dns_resolvers": {}
      }
    ]
  },
  "inherit_old_mosnconfig": true,
  "tracing": {},
  "metrics": {
    "sinks": null,
    "stats_matcher": {},
    "shm_zone": "",
    "shm_size": "0B"
  },
  "admin": {
    "address": {
      "socket_address": {
        "address": "0.0.0.0",
        "port_value": 34902
      }
    }
  },
  "pprof": {
    "debug": false,
    "port_value": 0
  },
  "plugin": {
    "log_base": ""
  }
}`
