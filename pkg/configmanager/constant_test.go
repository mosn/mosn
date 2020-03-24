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

// some test constant defines
var basicConfigStr = `
{
	"servers": [
		{
			"listeners": [
				{
					"name": "egress",
					"filter_chains": [
						{}
					],
					"stream_filters": [
						{
							"type":"test",
							"config": {
								"version": "1.0"
							}
						}
					]
				},
				{
					"name": "ingress",
					"filter_chains": [
						{}	
					]
				}
			],
			"routers": [
				{
					"router_config_name":"egress_router",
					"virtual_hosts":[
						{
							"name": "egress",
							"domains":["*"],
							"routers": [
								{
									"match": {"prefix":"/"},
									"route":{"cluster_name":"test1"}
								}
							]
						}
					]
				},
				{
					"router_config_name":"ingress_router",
					"virtual_hosts":[
						{
							"name": "ingress",									
							"domains":["*"],
							"routers": [
								{
									"match": {"prefix":"/"},
									"route":{"cluster_name":"test1"}
								}
							]
						}
					]
				}	
			]
		}
	],
	"cluster_manager":{
		"clusters":[
			{
				"name": "test_cluster",
				"type": "SIMPLE",
				"lb_type": "LB_RANDOM",
				"hosts": []
			}
		]
	}
}`
