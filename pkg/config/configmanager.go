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

import (
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
)

// dumper provides basic operation with mosn elements, like 'cluster', to write back the config file with dynamic changes
// biz logic operation, like 'clear all subscribe info', should be written in the bridge code, not in config module.
//
// changes dump flow :
//
// biz ops -> bridge module -> config module
//
//  dumped info load flow:
//
// 1. bridge module register key of interesting config(like 'cluster') into config module
// 2. config parser invoke callback functions (if exists) of config key
// 3. bridge module get biz info(like service subscribe/publish, application info) from callback invocations
// 4. biz module(like confreg) get biz info from bridge module directly

// ResetServiceRegistryInfo
// called when reset service registry info received
func ResetServiceRegistryInfo(appInfo v2.ApplicationInfo, subServiceList []string) {
	// reset service info
	config.ServiceRegistry.ServiceAppInfo = v2.ApplicationInfo{
		AntShareCloud: appInfo.AntShareCloud,
		DataCenter:    appInfo.DataCenter,
		AppName:       appInfo.AppName,
	}

	// reset servicePubInfo
	config.ServiceRegistry.ServicePubInfo = []v2.PublishInfo{}

	// delete subInfo / dynamic clusters
	removeClusterConfig(subServiceList)
}

// AddClusterConfig
// called when add cluster config info received
func AddClusterConfig(clusters []v2.Cluster) {
	for _, clusterConfig := range clusters {
		exist := false

		for i := range config.ClusterManager.Clusters {
			// rewrite cluster's info if exist already
			if config.ClusterManager.Clusters[i].Name == clusterConfig.Name {
				config.ClusterManager.Clusters[i] = clusterConfig
				exist = true
				break
			}
		}

		//added cluster if not exist
		if !exist {
			config.ClusterManager.Clusters = append(config.ClusterManager.Clusters, clusterConfig)
		}

		// update routes
		//AddRouterConfig(cluster.Name)
	}
	go dump(true)
}

func removeClusterConfig(clusterNames []string) {
	dirty := false

	for _, clusterName := range clusterNames {
		for i, cluster := range config.ClusterManager.Clusters {
			if cluster.Name == clusterName {
				//remove
				config.ClusterManager.Clusters = append(config.ClusterManager.Clusters[:i], config.ClusterManager.Clusters[i+1:]...)
				dirty = true
				break
			}
		}
	}

	go dump(dirty)
}

// AddPubInfo
// called when add pub info received
func AddPubInfo(pubInfoAdded map[string]string) {
	for srvName, srvData := range pubInfoAdded {
		exist := false
		srvPubInfo := v2.PublishInfo{
			Pub: v2.PublishContent{
				ServiceName: srvName,
				PubData:     srvData,
			},
		}
		for i := range config.ServiceRegistry.ServicePubInfo {
			// rewrite cluster's info
			if config.ServiceRegistry.ServicePubInfo[i].Pub.ServiceName == srvName {
				config.ServiceRegistry.ServicePubInfo[i] = srvPubInfo
				exist = true
				break
			}
		}

		if !exist {
			config.ServiceRegistry.ServicePubInfo = append(config.ServiceRegistry.ServicePubInfo, srvPubInfo)
		}
	}

	go dump(true)
}

// DelPubInfo
// called when delete publish info received
func DelPubInfo(serviceName string) {
	dirty := false

	for i, srvPubInfo := range config.ServiceRegistry.ServicePubInfo {
		if srvPubInfo.Pub.ServiceName == serviceName {
			//remove
			config.ServiceRegistry.ServicePubInfo = append(config.ServiceRegistry.ServicePubInfo[:i], config.ServiceRegistry.ServicePubInfo[i+1:]...)
			dirty = true
			break
		}
	}

	go dump(dirty)
}

// AddRouterConfig
// Add router from config when new cluster created
func AddRouterConfig(clusterName string) {
	routerName := clusterName[0 : len(clusterName)-8]

	for _, l := range config.Servers[0].Listeners {
		if routers, ok := l.FilterChains[0].Filters[0].Config["routes"].([]interface{}); ok {
			// remove repetition
			for _, route := range routers {
				if r, ok := route.(map[string]interface{}); ok {
					if n, ok := r["name"].(string); ok {
						if n == routerName {
							return
						}
					}
				}
			}

			// append router
			var s = make(map[string]interface{}, 4)
			s["name"] = routerName
			s["service"] = routerName
			s["cluster"] = clusterName
			routers = append(routers, s)
			l.FilterChains[0].Filters[0].Config["routes"] = routers
		} else {
			log.DefaultLogger.Println(l.FilterChains[0].Filters[0].Config["routes"])
		}
	}
}
