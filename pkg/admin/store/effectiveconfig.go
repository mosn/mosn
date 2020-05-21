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

package store

import (
	"encoding/json"
	"sync"

	"mosn.io/mosn/pkg/config/v2"
)

// effectiveConfig represents mosn's runtime config model
// MOSNConfig is the original config when mosn start
type effectiveConfig struct {
	MosnConfig       *v2.MOSNConfig                    `json:"mosn_config,omitempty"`
	Listener         map[string]v2.Listener            `json:"listener,omitempty"`
	Cluster          map[string]v2.Cluster             `json:"cluster,omitempty"`
	Routers          map[string]v2.RouterConfiguration `json:"routers,omitempty"`
	routerConfigPath map[string]string                 `json:"-"`
}

var conf effectiveConfig
var mutex sync.RWMutex

func init() {

	conf = effectiveConfig{
		Listener:         make(map[string]v2.Listener),
		Cluster:          make(map[string]v2.Cluster),
		Routers:          make(map[string]v2.RouterConfiguration),
		routerConfigPath: make(map[string]string),
	}
}

func Reset() {
	mutex.Lock()
	defer mutex.Unlock()
	conf.Listener = make(map[string]v2.Listener)
	conf.Cluster = make(map[string]v2.Cluster)
	conf.Routers = make(map[string]v2.RouterConfiguration)
	conf.routerConfigPath = make(map[string]string)
}

func SetMosnConfig(cfg *v2.MOSNConfig) {
	mutex.Lock()
	defer mutex.Unlock()
	conf.MosnConfig = &v2.MOSNConfig{}
	*conf.MosnConfig = *cfg
	// Clear the changed config
	conf.MosnConfig.ServiceRegistry = v2.ServiceRegistryInfo{}
	conf.MosnConfig.ClusterManager = v2.ClusterManagerConfig{}
	conf.MosnConfig.Servers = []v2.ServerConfig{}
	if len(cfg.Servers) > 0 {
		srv := cfg.Servers[0] // support only one server
		conf.MosnConfig.Servers = []v2.ServerConfig{
			{
				DefaultLogPath:  srv.DefaultLogPath,
				DefaultLogLevel: srv.DefaultLogLevel,
				GlobalLogRoller: srv.GlobalLogRoller,
				UseNetpollMode:  srv.UseNetpollMode,
				GracefulTimeout: srv.GracefulTimeout,
			},
		}
	}
}

// SetListenerConfig
// Set listener config when AddOrUpdateListener
func SetListenerConfig(listenerName string, listenerConfig v2.Listener) {
	mutex.Lock()
	defer mutex.Unlock()
	if config, ok := conf.Listener[listenerName]; ok {
		// mosn does not support update listener's network filter and stream filter for the time being
		// FIXME
		config.ListenerConfig.FilterChains = listenerConfig.ListenerConfig.FilterChains
		config.ListenerConfig.StreamFilters = listenerConfig.ListenerConfig.StreamFilters
		conf.Listener[listenerName] = config
	} else {
		conf.Listener[listenerName] = listenerConfig
	}
}

func SetClusterConfig(clusterName string, cluster v2.Cluster) {
	mutex.Lock()
	defer mutex.Unlock()
	conf.Cluster[clusterName] = cluster
	tryDump()
}

func RemoveClusterConfig(clusterName string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(conf.Cluster, clusterName)
	tryDump()
}

func SetHosts(clusterName string, hostConfigs []v2.Host) {
	mutex.Lock()
	defer mutex.Unlock()
	if cluster, ok := conf.Cluster[clusterName]; ok {
		cluster.Hosts = hostConfigs
		conf.Cluster[clusterName] = cluster
	}
	tryDump()
}

func SetRouter(routerName string, router v2.RouterConfiguration) {
	mutex.Lock()
	defer mutex.Unlock()
	// keep the origin router path for config persistent
	conf.routerConfigPath[routerName] = router.RouterConfigPath
	// clear the router's dynamic mode, so the dump api will show all routes in the router
	router.RouterConfigPath = ""
	conf.Routers[routerName] = router
	tryDump()
}

// Dump
// Dump all config
func Dump() ([]byte, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return json.Marshal(conf)
}

const (
	CfgTypeMOSN     = "MOSN"
	CfgTypeRouter   = "Router"
	CfgTypeCluster  = "Cluster"
	CfgTypeListener = "Listener"
)

func GetMOSNConfig(typ string) interface{} {
	mutex.RLock()
	defer mutex.RUnlock()
	switch typ {
	case CfgTypeMOSN:
		return conf.MosnConfig
	case CfgTypeRouter:
		return conf.Routers
	case CfgTypeCluster:
		return conf.Cluster
	case CfgTypeListener:
		return conf.Listener
	default:
		return nil
	}
}
