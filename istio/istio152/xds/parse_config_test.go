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

	"mosn.io/mosn/pkg/config/v2"
)

const xdsConfig = `
{
  "dynamic_resources": {
    "lds_config": {
      "ads": {}
    },
    "cds_config": {
      "ads": {}
    },
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
        "name": "prometheus_stats",
        "type": "STATIC",
        "connect_timeout": "0.250s",
        "lb_policy": "ROUND_ROBIN",
        "hosts": [
          {
            "socket_address": {
              "protocol": "TCP",
              "address": "127.0.0.1",
              "port_value": 15000
            }
          }
        ]
      },
      {
        "name": "sds-grpc",
        "type": "STATIC",
        "http2_protocol_options": {},
        "connect_timeout": "10s",
        "lb_policy": "ROUND_ROBIN",
        "hosts": [{
          "pipe": {
            "path": "/etc/istio/proxy/SDS"
           }
        }]
      },
      {
        "name": "xds-grpc",
        "type": "STRICT_DNS",
        "dns_refresh_rate": "300s",
        "dns_lookup_family": "V4_ONLY",
        "connect_timeout": "10s",
        "lb_policy": "ROUND_ROBIN",
        
        
        "tls_context": {
          "common_tls_context": {
            "alpn_protocols": [
              "h2"
            ],
            
             "tls_certificate_sds_secret_configs":[
              {
                "name":"default",
                "sds_config":{
                  "api_config_source":{
                    "api_type":"GRPC",
                    "grpc_services":[
                      {
                        "envoy_grpc":{
                          "cluster_name": "sds-grpc"
                        }
                      }
                    ]
                  }
                }
              }
            ],
            "validation_context": {
              "trusted_ca": {
                
                "filename": "./var/run/secrets/istio/root-cert.pem"
                
              },
              "verify_subject_alt_name": ["istiod.istio-system.svc"]
            }
            
          }
        },
        
        
        "hosts": [
          {
             "socket_address": {"address": "istio-pilot.istio-system.svc", "port_value": 15010}
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
        "max_requests_per_connection": 1,
        "http2_protocol_options": { }
      }
      
      ,
      {
        "name": "zipkin",
        "type": "STRICT_DNS",
        "dns_refresh_rate": "300s",
        "dns_lookup_family": "V4_ONLY",
        "connect_timeout": "1s",
        "lb_policy": "ROUND_ROBIN",
        "hosts": [
          {
            "socket_address": {"address": "zipkin.istio-system", "port_value": 9411}
          }
        ]
      }
    ]
  }
}`

func TestUnmarshalResources(t *testing.T) {
	cfg := v2.MOSNConfig{}
	if err := json.Unmarshal([]byte(xdsConfig), &cfg); err != nil {
		t.Errorf("unmarshal mosn config failed: %v", err)
	}
	xcfg, err := UnmarshalResources(cfg.RawDynamicResources, cfg.RawStaticResources)
	if err != nil {
		t.Errorf("UnmarshalResources error: %v", err)
	}
	_ = xcfg

}
