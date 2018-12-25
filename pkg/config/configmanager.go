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

// TODO: The functions in this file is for service discovery, but the function implmentation is not general, should fix it

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

// AddOrUpdateClusterConfig
// called when add cluster config info received
func AddOrUpdateClusterConfig(clusters []v2.Cluster) {
	addOrUpdateClusterConfig(clusters)
	go dump(true)
}

func addOrUpdateClusterConfig(clusters []v2.Cluster) {
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

// AddClusterWithRouter is a wrapper of AddOrUpdateCluster and AddRoutersConfig
// use this function to only dump config once
func AddClusterWithRouter(listenername string, virtualhost string, router v2.Router, clusters []v2.Cluster) {
	addOrUpdateClusterConfig(clusters)
	addRoutersConfig(listenername, virtualhost, router)
	go dump(true)
}

// AddRoutersConfig dumps addRoutersConfig result
func AddRoutersConfig(listenername string, virtualhost string, router v2.Router) {
	if addRoutersConfig(listenername, virtualhost, router) {
		go dump(true)
	}
}

// addRoutersConfig effects listener config.
// routers in listener.connection_manager.virtual_hosts, if a virtual host name is empty, use default virtual host
func addRoutersConfig(listenername string, virtualhost string, router v2.Router) bool {
	dirty := false
	// support only one server
	listeners := config.Servers[0].Listeners
	for _, ln := range listeners {
		if ln.Name != listenername {
			continue
		}
		// support only one filter chain
		nfs := ln.FilterChains[0].Filters
	FindRouter:
		for filterIndex, nf := range nfs {
			if nf.Type != v2.CONNECTION_MANAGER {
				continue FindRouter
			}
			routerConfiguration := &v2.RouterConfiguration{}
			if data, err := json.Marshal(nf.Config); err == nil {
				if err := json.Unmarshal(data, routerConfiguration); err != nil {
					log.DefaultLogger.Errorf("invalid router config, update config failed")
					break FindRouter
				}
			}
			var vhost *v2.VirtualHost
			vhs := routerConfiguration.VirtualHosts
		FindVirtualHost:
			for _, vh := range vhs {
				if virtualhost == "" {
					if len(vh.Domains) > 0 && vh.Domains[0] == "*" {
						vhost = vh
						break FindVirtualHost
					}
				} else if vh.Name == virtualhost {
					vhost = vh
					break FindVirtualHost
				}
			}
			if vhost != nil {
				vhost.Routers = append(vhost.Routers, router)
				dirty = true
			}
			if dirty {
				filterConfig := make(map[string]interface{})
				if data, err := json.Marshal(routerConfiguration); err == nil {
					if err := json.Unmarshal(data, &filterConfig); err != nil {
						log.DefaultLogger.Errorf("rewrite filter config failed")
						return false
					}
				}
				nfs[filterIndex] = v2.Filter{
					Type:   v2.CONNECTION_MANAGER,
					Config: filterConfig,
				}
			}
		}
	}
	return dirty
}
