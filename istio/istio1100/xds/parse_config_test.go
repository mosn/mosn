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
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/server"
	"mosn.io/mosn/pkg/upstream/cluster"
)

const xdsConfigStr = `{
	"dynamic_resources": {
		"lds_config": {
			"ads": {},
			"initial_fetch_timeout": "0s",
			"resource_api_version": "V3"
		},
		"cds_config": {
			"ads": {},
			"initial_fetch_timeout": "0s",
			"resource_api_version": "V3"
		},
		"ads_config": {
			"api_type": "GRPC",
			"set_node_on_first_message_only": true,
			"transport_api_version": "V3",
			"grpc_services": [
				{
					"envoy_grpc": {
						"cluster_name": "xds-grpc"
					}
				}
			]
		}
	},
"static_resources": {
    "clusters": [
      {
        "name": "prometheus_stats",
        "type": "STATIC",
        "connect_timeout": "0.250s",
        "lb_policy": "ROUND_ROBIN",
        "load_assignment": {
          "cluster_name": "prometheus_stats",
          "endpoints": [{
            "lb_endpoints": [{
              "endpoint": {
                "address":{
                  "socket_address": {
                    "protocol": "TCP",
                    "address": "127.0.0.1",
                    "port_value": 15000
                  }
                }
              }
            }]
          }]
        }
      },
      {
        "name": "agent",
        "type": "STATIC",
        "connect_timeout": "0.250s",
        "lb_policy": "ROUND_ROBIN",
        "load_assignment": {
          "cluster_name": "agent",
          "endpoints": [{
            "lb_endpoints": [{
              "endpoint": {
                "address":{
                  "socket_address": {
                    "protocol": "TCP",
                    "address": "127.0.0.1",
                    "port_value": 15020
                  }
                }
              }
            }]
          }]
        }
      },
      {
        "name": "sds-grpc",
        "type": "STATIC",
        "http2_protocol_options": {},
        "connect_timeout": "1s",
        "lb_policy": "ROUND_ROBIN",
        "load_assignment": {
          "cluster_name": "sds-grpc",
          "endpoints": [{
            "lb_endpoints": [{
              "endpoint": {
                "address":{
                  "pipe": {
                    "path": "./etc/istio/proxy/SDS"
                  }
                }
              }
            }]
          }]
        }
      },
      {
        "name": "xds-grpc",
        "type" : "STATIC",
        "connect_timeout": "1s",
        "lb_policy": "ROUND_ROBIN",
        "load_assignment": {
          "cluster_name": "xds-grpc",
          "endpoints": [{
            "lb_endpoints": [{
              "endpoint": {
                "address":{
                  "pipe": {
                    "path": "./etc/istio/proxy/XDS"
                  }
                }
              }
            }]
          }]
        },
        "circuit_breakers": {
          "thresholds": [
            {
              "priority": "DEFAULT",
              "max_connections": 100000,
              "max_pending_requests": 100000,
              "max_requests": 100000
            },
            {
              "priority": "HIGH",
              "max_connections": 100000,
              "max_pending_requests": 100000,
              "max_requests": 100000
            }
          ]
        },
        "upstream_connection_options": {
          "tcp_keepalive": {
            "keepalive_time": 300
          }
        },
        "max_requests_per_connection": 1,
        "http2_protocol_options": { }
      }
      
      ,
      {
        "name": "zipkin",
        "type": "STRICT_DNS",
        "respect_dns_ttl": true,
        "dns_lookup_family": "V4_ONLY",
        "dns_refresh_rate": "30s",
        "connect_timeout": "1s",
        "lb_policy": "ROUND_ROBIN",
        "load_assignment": {
          "cluster_name": "zipkin",
          "endpoints": [{
            "lb_endpoints": [{
              "endpoint": {
                "address":{
                  "socket_address": {"address": "zipkin.istio-system", "port_value": 9411}
                }
              }
            }]
          }]
        }
      }
      
      
    ],
    "listeners":[
      {
        "address": {
          "socket_address": {
            "protocol": "TCP",
            "address": "0.0.0.0",
            "port_value": 15090
          }
        },
        "filter_chains": [
          {
            "filters": [
              {
                "name": "envoy.filters.network.http_connection_manager",
                "typed_config": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                  "codec_type": "AUTO",
                  "stat_prefix": "stats",
                  "route_config": {
                    "virtual_hosts": [
                      {
                        "name": "backend",
                        "domains": [
                          "*"
                        ],
                        "routes": [
                          {
                            "match": {
                              "prefix": "/stats/prometheus"
                            },
                            "route": {
                              "cluster": "prometheus_stats"
                            }
                          }
                        ]
                      }
                    ]
                  },
                  "http_filters": [{
                    "name": "envoy.filters.http.router",
                    "typed_config": {
                      "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
                    }
                  }]
                }
              }
            ]
          }
        ]
      },
      {
        "address": {
          "socket_address": {
            "protocol": "TCP",
            "address": "0.0.0.0",
            "port_value": 15021
          }
        },
        "filter_chains": [
          {
            "filters": [
              {
                "name": "envoy.filters.network.http_connection_manager",
                "typed_config": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                  "codec_type": "AUTO",
                  "stat_prefix": "agent",
                  "route_config": {
                    "virtual_hosts": [
                      {
                        "name": "backend",
                        "domains": [
                          "*"
                        ],
                        "routes": [
                          {
                            "match": {
                              "prefix": "/healthz/ready"
                            },
                            "route": {
                              "cluster": "agent"
                            }
                          }
                        ]
                      }
                    ]
                  },
                  "http_filters": [{
                    "name": "envoy.filters.http.router",
                    "typed_config": {
                      "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
                    }
                  }]
                }
              }
            ]
          }
        ]
      }
    ]
  }
}`

func TestMain(m *testing.M) {
	// init
	router.NewRouterManager()
	cm := cluster.NewClusterManagerSingleton(nil, nil, nil)
	sc := server.NewConfig(&v2.ServerConfig{
		ServerName:      "test_xds_server",
		DefaultLogPath:  "stdout",
		DefaultLogLevel: "FATAL",
	})
	server.NewServer(sc, &mockCMF{}, cm)
	os.Exit(m.Run())
}

func TestUnmarshalResources(t *testing.T) {
	cfg := &v2.MOSNConfig{}
	err := json.Unmarshal([]byte(xdsConfigStr), cfg)
	require.Nil(t, err)
	xdsConfig, err := UnmarshalResources(cfg.RawDynamicResources, cfg.RawStaticResources)
	require.Nil(t, err)
	require.NotNil(t, xdsConfig)
	// verify cluster and listener
	listenerAdapter := server.GetListenerAdapterInstance()
	require.NotNil(t, listenerAdapter.FindListenerByName("", "127.0.0.1_15090"))
	require.NotNil(t, listenerAdapter.FindListenerByName("", "127.0.0.1_15021"))
	clusterAdapter := cluster.GetClusterMngAdapterInstance()
	for _, cn := range []string{
		"prometheus_stats",
		"agent",
		"sds-grpc",
		"xds-grpc",
		// zipkin is ignored.
	} {
		require.True(t, clusterAdapter.ClusterExist(cn))
	}

}
