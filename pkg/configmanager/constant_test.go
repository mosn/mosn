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
	"fmt"
	"io/ioutil"
	"os"
)

const mosnConfig = `{
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
                                                                                 "downstream_protocol": "X",
                                                                                 "upstream_protocol": "X",
										 "extend_config": {
											 "sub_protocols":"bolt,boltv2"
										 },
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
                                 "address": "127.0.0.1",
                                 "port_value": 34901
                         }
                 }
         },
         "metrics": {
                 "stats_matcher": {
                         "reject_all": true
                 }
         },
	 "tracing": {
		 "enable": true,
		 "driver": "SOFATracer"
	 },
	 "pprof": {
		 "debug": true,
		 "port_value": 34902
	 }
}`

const testConfigPath = "/tmp/mosn_admin.json"

func createMosnConfig() {
	routerPath := "/tmp/routers/test_router/"
	clusterPath := "/tmp/clusters"

	os.Remove(testConfigPath)
	os.RemoveAll(clusterPath)
	os.MkdirAll(clusterPath, 0755)
	os.RemoveAll(routerPath)
	os.MkdirAll(routerPath, 0755)

	ioutil.WriteFile(testConfigPath, []byte(mosnConfig), 0644)
	// write router
	ioutil.WriteFile(fmt.Sprintf("%s/virtualhost_0.json", routerPath), []byte(`{
                "name": "virtualhost_0"
        }`), 0644)
	// write cluster
	ioutil.WriteFile(fmt.Sprintf("%s/cluster001.json", clusterPath), []byte(`{
                "name": "cluster001",
                "type": "SIMPLE",
                "lb_type": "LB_RANDOM"
        }`), 0644)

}
