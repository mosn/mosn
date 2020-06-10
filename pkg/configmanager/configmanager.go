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
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

// AddOrUpdateClusterConfig
// called when add cluster config info received
func AddOrUpdateClusterConfig(clusters []v2.Cluster) {
	addOrUpdateClusterConfig(clusters)
	dump(true)
}

func addOrUpdateClusterConfig(clusters []v2.Cluster) {
	configLock.Lock()
	defer configLock.Unlock()
	for _, clusterConfig := range clusters {
		exist := false

		for i := range config.ClusterManager.Clusters {
			// rewrite cluster's info if exist already
			if config.ClusterManager.Clusters[i].Name == clusterConfig.Name {
				config.ClusterManager.Clusters[i] = clusterConfig
				if log.DefaultLogger.GetLogLevel() >= log.INFO {
					log.DefaultLogger.Infof("[configmanager] [update cluster] update cluster %s", clusterConfig.Name)
				}
				exist = true
				break
			}
		}

		//added cluster if not exist
		if !exist {
			if log.DefaultLogger.GetLogLevel() >= log.INFO {
				log.DefaultLogger.Infof("[configmanager] [add cluster] add cluster %s", clusterConfig.Name)
			}
			config.ClusterManager.Clusters = append(config.ClusterManager.Clusters, clusterConfig)
		}
	}
}

func RemoveClusterConfig(clusterNames []string) {
	if removeClusterConfig(clusterNames) {
		dump(true)
	}
}

func removeClusterConfig(clusterNames []string) bool {
	configLock.Lock()
	defer configLock.Unlock()
	dirty := false
	for _, clusterName := range clusterNames {
		for i, cluster := range config.ClusterManager.Clusters {
			if cluster.Name == clusterName {
				//remove
				config.ClusterManager.Clusters = append(config.ClusterManager.Clusters[:i], config.ClusterManager.Clusters[i+1:]...)
				if log.DefaultLogger.GetLogLevel() >= log.INFO {
					log.DefaultLogger.Infof("[configmanager] [remove cluster] remove cluster %s", clusterName)
				}
				dirty = true
				break
			}
		}
	}
	return dirty
}

func DeleteClusterHost(clusterName string, hostaddr string) {
	configLock.Lock()
	defer configLock.Unlock()
	dirty := false
FoundCluster:
	for i, cluster := range config.ClusterManager.Clusters {
		if cluster.Name == clusterName {
			// found host
			foundIdx := -1
			for idx, host := range cluster.Hosts {
				if host.Address == hostaddr {
					foundIdx = idx
					break
				}
			}
			if foundIdx != -1 {
				cluster.Hosts = append(cluster.Hosts[:foundIdx], cluster.Hosts[foundIdx+1:]...)
				config.ClusterManager.Clusters[i] = cluster
				dirty = true
				break FoundCluster
			}
		}
	}
	dump(dirty)
}

func AddOrUpdateClusterHost(clusterName string, hostCfg v2.Host) {
	configLock.Lock()
	defer configLock.Unlock()
FoundCluster:
	for i, cluster := range config.ClusterManager.Clusters {
		if cluster.Name == clusterName {
			for idx, host := range cluster.Hosts {
				if host.Address == hostCfg.Address {
					// update
					cluster.Hosts[idx] = hostCfg
					config.ClusterManager.Clusters[i] = cluster
					break FoundCluster
				}
			}
			// add a new host
			cluster.Hosts = append(cluster.Hosts, hostCfg)
			config.ClusterManager.Clusters[i] = cluster
			break FoundCluster
		}
	}
	dump(true)
}

// AddPubInfo
// called when add pub info received
func AddPubInfo(pubInfoAdded map[string]string) {
	configLock.Lock()
	defer configLock.Unlock()
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

	dump(true)
}

// DelPubInfo
// called when delete publish info received
func DelPubInfo(serviceName string) {
	configLock.Lock()
	defer configLock.Unlock()
	dirty := false

	for i, srvPubInfo := range config.ServiceRegistry.ServicePubInfo {
		if srvPubInfo.Pub.ServiceName == serviceName {
			//remove
			config.ServiceRegistry.ServicePubInfo = append(config.ServiceRegistry.ServicePubInfo[:i], config.ServiceRegistry.ServicePubInfo[i+1:]...)
			dirty = true
			break
		}
	}

	dump(dirty)
}

// AddClusterWithRouter is a wrapper of AddOrUpdateCluster and AddOrUpdateRoutersConfig
// use this function to only dump config once
func AddClusterWithRouter(clusters []v2.Cluster, routerConfig *v2.RouterConfiguration) {
	addOrUpdateClusterConfig(clusters)
	addOrUpdateRouterConfig(routerConfig)
	dump(true)
}

func findListener(listenername string) (v2.Listener, int) {
	// support only one server
	listeners := config.Servers[0].Listeners
	for idx, ln := range listeners {
		if ln.Name == listenername {
			return ln, idx
		}
	}
	return v2.Listener{}, -1
}

// AddOrUpdateRouterConfig update the router config
func AddOrUpdateRouterConfig(routerConfig *v2.RouterConfiguration) {
	if addOrUpdateRouterConfig(routerConfig) {
		dump(true)
	}
}

func addOrUpdateRouterConfig(routerConfig *v2.RouterConfiguration) bool {
	configLock.Lock()
	defer configLock.Unlock()
	if routerConfig == nil {
		return false
	}
	// support only one server
	routers := config.Servers[0].Routers
	for idx, rt := range routers {
		if rt.RouterConfigName == routerConfig.RouterConfigName {
			config.Servers[0].Routers[idx] = routerConfig
			return true
		}
	}
	// not equal
	config.Servers[0].Routers = append(config.Servers[0].Routers, routerConfig)
	return true
}

func AddOrUpdateListener(listener *v2.Listener) {
	configLock.Lock()
	defer configLock.Unlock()

	if listener == nil {
		return
	}

	_, idx := findListener(listener.Name)
	if idx == -1 {
		config.Servers[0].Listeners = append(config.Servers[0].Listeners, *listener)
	} else {
		config.Servers[0].Listeners[idx] = *listener
	}
	dump(true)
}

// FIXME: all config should be changed to pointer instead of struct
func UpdateFullConfig(listeners []v2.Listener, routers []*v2.RouterConfiguration, clusters []v2.Cluster) {
	configLock.Lock()
	defer configLock.Unlock()

	config.Servers[0].Listeners = listeners
	config.Servers[0].Routers = routers
	config.ClusterManager.Clusters = clusters

	dump(true)
}

// DEPRECATED: we will remove these functions, because we will use a general extendable field to instead of these fields.
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
	configLock.Lock()
	// reset service info
	config.ServiceRegistry.ServiceAppInfo = v2.ApplicationInfo{
		AntShareCloud:    appInfo.AntShareCloud,
		DataCenter:       appInfo.DataCenter,
		AppName:          appInfo.AppName,
		DeployMode:       appInfo.DeployMode,
		MasterSystem:     appInfo.MasterSystem,
		CloudName:        appInfo.CloudName,
		HostMachine:      appInfo.HostMachine,
		InstanceId:       appInfo.InstanceId,
		RegistryEndpoint: appInfo.RegistryEndpoint,
		AccessKey:        appInfo.AccessKey,
		SecretKey:        appInfo.SecretKey,
	}

	// reset servicePubInfo
	config.ServiceRegistry.ServicePubInfo = []v2.PublishInfo{}
	configLock.Unlock()

	// delete subInfo / dynamic clusters
	RemoveClusterConfig(subServiceList)
}

// AddMsgMeta
// called when msg meta updated
func AddMsgMeta(dataId, groupId string) {
	configLock.Lock()
	defer configLock.Unlock()
	if config.ServiceRegistry.MsgMetaInfo == nil {
		config.ServiceRegistry.MsgMetaInfo = make(map[string][]string)
	}

	groupIds, ok := config.ServiceRegistry.MsgMetaInfo[dataId]
	if !ok {
		groupIds = make([]string, 0, 8)
		config.ServiceRegistry.MsgMetaInfo[dataId] = groupIds
	}

	exist := false
	for i := range groupIds {
		if groupIds[i] == groupId {
			exist = true
			break
		}
	}

	if !exist {
		config.ServiceRegistry.MsgMetaInfo[dataId] = append(config.ServiceRegistry.MsgMetaInfo[dataId], groupId)
	}

	dump(true)
}

// DelMsgMeta
// called when delete msg meta received
func DelMsgMeta(dataId string) {
	configLock.Lock()
	defer configLock.Unlock()
	dirty := false

	if _, ok := config.ServiceRegistry.MsgMetaInfo[dataId]; ok {
		delete(config.ServiceRegistry.MsgMetaInfo, dataId)
		dirty = true
	}

	dump(dirty)
}

// UpdateMqClientKey update mq client registry info
func UpdateMqClientKey(id, clientKey string, remove bool) {
	configLock.Lock()
	defer configLock.Unlock()
	if config.ServiceRegistry.MqClientKey == nil {
		config.ServiceRegistry.MqClientKey = make(map[string]string)
	}

	if remove {
		delete(config.ServiceRegistry.MqClientKey, id)
	} else {
		config.ServiceRegistry.MqClientKey[id] = clientKey
	}

	dump(true)
}

// UpdteMqMeta update mq meta info
func UpdateMqMeta(topic, meta string, remove bool) {
	configLock.Lock()
	defer configLock.Unlock()
	if config.ServiceRegistry.MqMeta == nil {
		config.ServiceRegistry.MqMeta = make(map[string]string)
	}

	if remove {
		delete(config.ServiceRegistry.MqMeta, topic)
	} else {
		config.ServiceRegistry.MqMeta[topic] = meta
	}

	dump(true)
}

// SetMqConsumers update topic consumer list
func SetMqConsumers(key string, consumers []string) {
	configLock.Lock()
	defer configLock.Unlock()

	if config.ServiceRegistry.MqConsumers == nil {
		config.ServiceRegistry.MqConsumers = make(map[string][]string)
	}

	if len(key) != 0 {
		if len(consumers) != 0 {
			config.ServiceRegistry.MqConsumers[key] = consumers
			return
		}

		delete(config.ServiceRegistry.MqConsumers, key)
	}

	dump(true)
}

// RmMqConsumers remove topic consumer list
func RmMqConsumers(key string) {
	configLock.Lock()
	defer configLock.Unlock()
	if config.ServiceRegistry.MqConsumers == nil {
		config.ServiceRegistry.MqConsumers = make(map[string][]string)
		return
	}

	if len(key) != 0 {
		delete(config.ServiceRegistry.MqConsumers, key)
	}

	dump(true)
}
