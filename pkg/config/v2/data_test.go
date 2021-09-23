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
