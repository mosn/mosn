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

func AddRouterAndClusterConfigAndDump(clusters []v2.Cluster, clusterName string, matchName string) {
	AddClusterConfig(clusters)
	AddRouterConfig(clusterName, matchName)
	go dump(true)
}

func AddClusterConfigAndDump(clusters []v2.Cluster) {
	AddClusterConfig(clusters)
	go dump(true)
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

func AddRouterConfigAndDump(clusterName string, matchName string) {
	AddRouterConfig(clusterName, matchName)
	go dump(true)

}

// AddRouterConfig
// Add router from config when new cluster created
func AddRouterConfig(clusterName string, matchName string) {
	matchValue := clusterName

	l := config.Servers[0].Listeners[0]
	vhosts := l.FilterChains[0].Filters[1].Config["virtual_hosts"]

	log.DefaultLogger.Infof("AddRouterConfig start, clusterName=%s,matchName=%s", clusterName, matchName)

	if virtual_hosts, ok := vhosts.([]interface{}); ok {
		if vh, ok := virtual_hosts[0].(map[string]interface{}); ok {
			allRouters := vh["routers"]
			if isExistRouterConfig(allRouters, matchName, matchValue) {
				return
			}
			newRouter := make(map[string]interface{})
			matchMap := make(map[string]interface{})
			newRouter["match"] = matchMap

			matchHeaders := make([]interface{}, 0)
			header := make(map[string]interface{})
			header["name"] = matchName
			header["regex"] = false
			header["value"] = matchValue
			matchHeaders = append(matchHeaders, header)
			matchMap["headers"] = matchHeaders

			routerMap := make(map[string]interface{})
			newRouter["route"] = routerMap
			routerMap["cluster_name"] = matchValue

			if list, ok := allRouters.([]interface{}); ok {
				list := append(list, newRouter)
				vh["routers"] = list
			}
			log.DefaultLogger.Infof("AddRouterConfig success, clusterName=%s,matchName=%s", clusterName, matchName)
		} else {
			log.DefaultLogger.Errorf("AddRouterConfig error,first vhost is not map. vhosts=%+v", vhosts)
		}
	} else {
		log.DefaultLogger.Errorf("AddRouterConfig error,vhosts is not interface array. vhosts=%+v", vhosts)
	}
}

func isExistRouterConfig(routersConfig interface{}, matchName string, matchValue string) bool {
	if routers, ok := routersConfig.([]interface{}); ok {
		for _, routerConfig := range routers {
			if router, ok := routerConfig.(map[string]interface{}); ok {
				matchConfig := router["match"]
				if match, ok := matchConfig.(map[string]interface{}); ok {
					headersConfig := match["headers"]
					if headers, ok := headersConfig.([]interface{}); ok {
						headerConfig := headers[0]
						if header, ok := headerConfig.(map[string]interface{}); ok {
							if matchName == header["name"] && matchValue == header["value"] {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}
