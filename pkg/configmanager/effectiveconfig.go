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
)

// effectiveConfig represents mosn's runtime config model
// MOSNConfig is the original config when mosn start
// Sometimes we needs dump the config without some fields, so we use struct as the map value,
// so we can change the value and take no effects on the original data
type effectiveConfig struct {
	MosnConfig        v2.MOSNConfig                     `json:"mosn_config,omitempty"`
	Listener          map[string]v2.Listener            `json:"listener,omitempty"`
	Cluster           map[string]v2.Cluster             `json:"cluster,omitempty"`
	Routers           map[string]v2.RouterConfiguration `json:"routers,omitempty"`
	ExtendConfigs     []v2.ExtendConfig                 `json:"extends,omitempty"`
	clusterConfigPath string                            `json:"-"`
	routerConfigPath  map[string]string                 `json:"-"`
}

func init() {
	conf = effectiveConfig{
		Listener:         make(map[string]v2.Listener),
		Cluster:          make(map[string]v2.Cluster),
		Routers:          make(map[string]v2.RouterConfiguration),
		ExtendConfigs:    []v2.ExtendConfig{},
		routerConfigPath: make(map[string]string),
	}
}

func Reset() {
	configLock.Lock()
	defer configLock.Unlock()
	conf.MosnConfig = v2.MOSNConfig{}
	conf.Listener = make(map[string]v2.Listener)
	conf.Cluster = make(map[string]v2.Cluster)
	conf.Routers = make(map[string]v2.RouterConfiguration)
	conf.ExtendConfigs = []v2.ExtendConfig{}
	conf.routerConfigPath = make(map[string]string)
}

func SetMosnConfig(cfg *v2.MOSNConfig) {
	if cfg == nil {
		return
	}
	configLock.Lock()
	defer configLock.Unlock()
	conf.MosnConfig = *cfg
	// Clear the changed config
	conf.MosnConfig.ClusterManager = v2.ClusterManagerConfig{
		ClusterManagerConfigJson: v2.ClusterManagerConfigJson{
			TLSContext: cfg.ClusterManager.TLSContext,
		},
	}
	conf.clusterConfigPath = cfg.ClusterManager.ClusterConfigPath // cluster config path should be stored
	conf.MosnConfig.Extends = nil
	// support only one server
	conf.MosnConfig.Servers = make([]v2.ServerConfig, 1)
	if len(cfg.Servers) > 0 {
		conf.MosnConfig.Servers[0] = cfg.Servers[0]
	}
	conf.MosnConfig.Servers[0].Listeners = nil
	conf.MosnConfig.Servers[0].Routers = nil
}

// SetListenerConfig update the listener config when AddOrUpdateListener
func SetListenerConfig(listenerConfig v2.Listener) {
	configLock.Lock()
	defer configLock.Unlock()
	listenerName := listenerConfig.Name
	conf.Listener[listenerName] = listenerConfig
	tryDump()
}

// SetClusterConfig update the cluster config when AddOrUpdateCluster
func SetClusterConfig(cluster v2.Cluster) {
	configLock.Lock()
	defer configLock.Unlock()
	conf.Cluster[cluster.Name] = cluster
	tryDump()
}

// SetRemoveClusterConfig update the cluster config when DeleteCluster
func SetRemoveClusterConfig(clusterName string) {
	configLock.Lock()
	defer configLock.Unlock()
	delete(conf.Cluster, clusterName)
	tryDump()
}

func SetHosts(clusterName string, hostConfigs []v2.Host) {
	configLock.Lock()
	defer configLock.Unlock()
	if cluster, ok := conf.Cluster[clusterName]; ok {
		cluster.Hosts = hostConfigs
		conf.Cluster[clusterName] = cluster
		tryDump()
	}
}

func SetRouter(router v2.RouterConfiguration) {
	configLock.Lock()
	defer configLock.Unlock()
	routerName := router.RouterConfigName
	// keep the origin router path for config persistent
	conf.routerConfigPath[routerName] = router.RouterConfigPath
	// clear the router's dynamic mode, so the dump api will show all routes in the router
	router.RouterConfigPath = ""
	conf.Routers[routerName] = router
	tryDump()
}

func SetExtend(typ string, cfg json.RawMessage) {
	configLock.Lock()
	defer configLock.Unlock()
	found := false
	for idx, ext := range conf.ExtendConfigs {
		if ext.Type == typ {
			ext.Config = cfg
			conf.ExtendConfigs[idx] = ext
			found = true
			break
		}
	}
	if !found {
		conf.ExtendConfigs = append(conf.ExtendConfigs, v2.ExtendConfig{
			Type:   typ,
			Config: cfg,
		})
	}
	tryDump()
}

func SetClusterManagerTLS(tls v2.TLSConfig) {
	configLock.Lock()
	defer configLock.Unlock()
	conf.MosnConfig.ClusterManager.TLSContext = tls
	tryDump()
}

// DumpJSON marshals the effectiveConfig to bytes
func DumpJSON() ([]byte, error) {
	configLock.RLock()
	defer configLock.RUnlock()
	return json.Marshal(conf)
}

const (
	CfgTypeMOSN     = "MOSN"
	CfgTypeRouter   = "Router"
	CfgTypeCluster  = "Cluster"
	CfgTypeListener = "Listener"
	CfgTypeExtend   = "Extend"
)

func getMOSNConfig(typ string) interface{} {
	switch typ {
	case CfgTypeMOSN:
		return conf.MosnConfig
	case CfgTypeRouter:
		return conf.Routers
	case CfgTypeCluster:
		return conf.Cluster
	case CfgTypeListener:
		return conf.Listener
	case CfgTypeExtend:
		return conf.ExtendConfigs
	default:
		return nil
	}
}

func HandleMOSNConfig(typ string, handle func(interface{})) {
	if handle == nil {
		return
	}
	configLock.RLock()
	defer configLock.RUnlock()
	v := getMOSNConfig(typ)
	handle(v)
}
