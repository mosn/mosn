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

package config

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
								},
								{
									 "type": "connection_manager",
									 "config": {
										 "router_config_name": "test_router",
										 "router_configs": "/tmp/routers/test_router/"
									 }
								}
							]
						}
					 ],
					 "stream_filters": [
					 ]
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

const xdsSdsConfig = `{
  "servers": [
    {
      "mosn_server_name": "mosn",
      "default_log_path": "/tmp/mosn/default.log",
      "default_log_level": "ERROR",
      "global_log_roller": "time=1",
      "graceful_timeout": "30s",
      "processor": 2,
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
                },
                {
                  "type": "connection_manager",
                  "config": {
                    "router_config_name": "sofa_egress_bolt_router",
                    "router_configs": "/tmp/mosn/routers/sofa_egress_bolt_router/"
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
                },
                {
                  "type": "connection_manager",
                  "config": {
                    "router_config_name": "sofa_ingress_bolt_router",
                    "router_configs": "/tmp/mosn/routers/sofa_ingress_bolt_router/"
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
  ]
}
`
