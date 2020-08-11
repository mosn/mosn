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
	"encoding/json"

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

func UpdateClusterManagerTLS(tls v2.TLSConfig) {
	configLock.Lock()
	defer configLock.Unlock()
	config.ClusterManager.TLSContext = tls
	dump(true)
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

	// easy for test (config.Servers maybe equals nil),
	// for example, a registry push does not trigger the dump store
	if config.Servers == nil {
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
func UpdateFullConfig(listeners []v2.Listener, routers []*v2.RouterConfiguration, clusters []v2.Cluster, extends map[string]json.RawMessage) {
	configLock.Lock()
	defer configLock.Unlock()

	config.Servers[0].Listeners = listeners
	config.Servers[0].Routers = routers
	config.ClusterManager.Clusters = clusters
	config.Extends = extends

	dump(true)
}

func UpdateExtendConfig(typ string, cfg json.RawMessage) {
	configLock.Lock()
	defer configLock.Unlock()
	if config.Extends == nil {
		config.Extends = map[string]json.RawMessage{}
	}
	config.Extends[typ] = cfg
	dump(true)
}
