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

const cfgStr = `{
	 "servers": [
	 	{
			"mosn_server_name": "test_mosn_server",
			"listeners": [
				{
					 "name": "test_mosn_listener",
					 "address": "127.0.0.1:8080",
					 "filter_chains": [
					 	{
							"filters": [
								{
									 "type": "proxy",
									 "config": {
										 "downstream_protocol": "SofaRpc",
										 "upstream_protocol": "SofaRpc",
										 "router_config_name": "test_router"
									 }
								}
							]
						}
					 ],
					 "stream_filters": [
					 ]
				}
			],
			"routers": [
				{
					"router_config_name": "test_router",
					"router_configs": "/tmp/routers/test_router/"
				}
			]
		}
	 ],
	 "cluster_manager": {
		 "clusters_configs": "/tmp/clusters"
	 },
	 "admin": {
		 "address": {
			 "socket_address": {
				 "address": "0.0.0.0",
				 "port_value": 34901
			 }
		 }
	 }
}`

// apiType = "GRPC" Defined by const ApiConfigSource_GRPC ApiConfigSource_ApiType = 2
// at the go-control-plane/envoy/api/v2/core/config_source.pb.go file.
const xdsSdsConfig = `{
  "servers": [
    {
      "mosn_server_name": "mosn",
      "default_log_path": "/tmp/mosn/default.log",
      "default_log_level": "ERROR",
      "global_log_roller": "time=1",
      "graceful_timeout": "30s",
      "processor": 2,
      "routers": [
      ],
      "listeners": [
        {
          "name": "egress_sofa_bolt",
          "type": "egress",
          "address": "0.0.0.0:12220",
          "bind_port": true,
          "access_logs": [
            {
              "log_path": "/tmp/mosn/access_egress.log",
              "log_format": "%StartTime% %RequestReceivedDuration% %ResponseReceivedDuration% %REQ.requestid% %REQ.cmdcode% %RESP.requestid% %RESP.service%"
            }
          ],
          "filter_chains": [
            {
              "tls_context_set": [
                {}
              ],
              "filters": [
                {
                  "type": "proxy",
                  "config": {
                    "downstream_protocol": "SofaRpc",
                    "name": "proxy_config",
                    "router_config_name": "sofa_egress_bolt_router",
                    "upstream_protocol": "SofaRpc"
                  }
                }
              ]
            }
          ],
          "stream_filters": [
            {
              "type": "healthcheck",
              "config": {
                "cache_time": "360s",
                "cluster_min_healthy_percentages": {
                  "local_service": 70
                },
                "passthrough": false
              }
            }
          ]
        },
        {
          "name": "ingress_sofa_bolt",
          "type": "ingress",
          "address": "0.0.0.0:12200",
          "bind_port": true,
          "access_logs": [
            {
              "log_path": "/tmp/mosn/access_ingress.log",
              "log_format": "%StartTime% %RequestReceivedDuration% %ResponseReceivedDuration% %REQ.requestid% %REQ.cmdcode% %RESP.requestid% %RESP.service%"
            }
          ],
          "filter_chains": [
            {
              "tls_context_set": [
                {
                  "status": true,
                  "verify_client": true,
                  "min_version": "TLS_AUTO",
                  "max_version": "TLS_AUTO",
                  "alpn": "h2,http/1.1",
                  "sds_source": {
                    "CertificateConfig": {
                      "name": "default",
                      "sdsConfig": {
                        "apiConfigSource": {
                          "apiType": "GRPC",
                          "grpcServices": [
                            {
                              "googleGrpc": {
                                "targetUri": "/var/run/sds",
                                "channelCredentials": {
                                  "localCredentials": {}
                                },
                                "callCredentials": [
                                  {
                                    "googleComputeEngine": {}
                                  }
                                ],
                                "statPrefix": "sdsstat"
                              }
                            }
                          ]
                        }
                      }
                    },
                    "ValidationConfig": {
                      "name": "ROOTCA",
                      "sdsConfig": {
                        "apiConfigSource": {
                          "apiType": "GRPC",
                          "grpcServices": [
                            {
                              "googleGrpc": {
                                "targetUri": "/var/run/sds",
                                "channelCredentials": {
                                  "localCredentials": {}
                                },
                                "callCredentials": [
                                  {
                                    "googleComputeEngine": {}
                                  }
                                ],
                                "statPrefix": "sdsstat"
                              }
                            }
                          ]
                        }
                      }
                    }
                  }
                }
              ],
              "filters": [
                {
                  "type": "proxy",
                  "config": {
                    "downstream_protocol": "SofaRpc",
                    "name": "proxy_config",
                    "router_config_name": "sofa_ingress_bolt_router",
                    "upstream_protocol": "SofaRpc"
                  }
                }
              ]
            }
          ],
          "stream_filters": [
            {
              "type": "healthcheck",
              "config": {
                "cache_time": "360s",
                "cluster_min_healthy_percentages": {
                  "local_service": 70
                },
                "passthrough": false
              }
            }
          ],
          "inspector": true
         }]
    }
  ],
  "dynamic_resources": {
    "ads_config": {
      "api_type": "GRPC",
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
        "name": "xds-grpc",
        "type": "STRICT_DNS",
        "connect_timeout": "10s",
        "lb_policy": "ROUND_ROBIN",
        "hosts": [
          {
            "socket_address": {"address": "pilot.test", "port_value": 15010}
          }
        ],
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
        "http2_protocol_options": { }
      }
    ]
  }

}
`
