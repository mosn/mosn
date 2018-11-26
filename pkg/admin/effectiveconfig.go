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

package admin

import (
	"sync"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
)

type effectiveConfig struct {
	MOSNConfig interface{}            `json:"mosn_config,omitempty"`
	Listener   map[string]v2.Listener `json:"listener,omitempty"`
	Cluster    map[string]v2.Cluster  `json:"cluster,omitempty"`
}

var (
	conf = effectiveConfig{
		Listener: make(map[string]v2.Listener),
		Cluster:  make(map[string]v2.Cluster),
	}
	mutex = new(sync.RWMutex)
)

func Reset() {
	mutex.Lock()
	conf.MOSNConfig = nil
	conf.Listener = make(map[string]v2.Listener)
	conf.Cluster = make(map[string]v2.Cluster)
	mutex.Unlock()
}

func SetMOSNConfig(msonConfig interface{}) {
	mutex.Lock()
	conf.MOSNConfig = msonConfig
	mutex.Unlock()
}

// SetListenerConfig
// Set listener config when AddOrUpdateListener
func SetListenerConfig(listenerName string, listenerConfig v2.Listener) {
	mutex.Lock()
	if config, ok := conf.Listener[listenerName]; ok {
		config.ListenerConfig.FilterChains = listenerConfig.ListenerConfig.FilterChains
		config.ListenerConfig.StreamFilters = listenerConfig.ListenerConfig.StreamFilters
		conf.Listener[listenerName] = config
	} else {
		conf.Listener[listenerName] = listenerConfig
	}
	mutex.Unlock()
}

func SetClusterConfig(clusterName string, cluster v2.Cluster) {
	mutex.Lock()
	conf.Cluster[clusterName] = cluster
	mutex.Unlock()
}

func SetHosts(clusterName string, hostConfigs []v2.Host) {
	mutex.Lock()
	if cluster, ok := conf.Cluster[clusterName]; ok {
		cluster.Hosts = hostConfigs
		conf.Cluster[clusterName] = cluster
	}
	mutex.Unlock()
}

// Dump
// Dump all config
func Dump() ([]byte, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return json.Marshal(conf)
}
