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

var (
	mutex           sync.RWMutex
	effectiveConfig map[string]interface{}
)

func init() {
	Reset()
}

func Reset() {
	mutex.Lock()
	effectiveConfig = make(map[string]interface{})
	effectiveConfig["listener"] = make(map[string]*v2.Listener)
	effectiveConfig["cluster"] = make(map[string]*v2.Cluster)
	effectiveConfig["original_config"] = nil
	mutex.Unlock()
}

func Set(key string, val interface{}) {
	mutex.Lock()
	effectiveConfig[key] = val
	mutex.Unlock()
}

func SetListenerConfig(listenerName string, listenerConfig *v2.Listener) {
	mutex.Lock()
	listenerConfigMap := effectiveConfig["listener"].(map[string]*v2.Listener)
	if originalConf, ok := listenerConfigMap[listenerName]; ok {
		originalConf.ListenerConfig.FilterChains = listenerConfig.ListenerConfig.FilterChains
		originalConf.ListenerConfig.StreamFilters = listenerConfig.ListenerConfig.StreamFilters
	} else {
		listenerConfigMap[listenerName] = listenerConfig
	}
	mutex.Unlock()
}

func SetClusterConfig(clusterName string, clusterConfig *v2.Cluster) {
	mutex.Lock()
	clusterConfigMap := effectiveConfig["cluster"].(map[string]*v2.Cluster)
	clusterConfigMap[clusterName] = clusterConfig
	mutex.Unlock()
}

func SetHosts(clusterName string, hosts []v2.Host) {
	mutex.Lock()
	clusterConfigMap := effectiveConfig["cluster"].(map[string]*v2.Cluster)
	if clusterConfig, ok := clusterConfigMap[clusterName]; ok {
		clusterConfig.Hosts = hosts
	}
	mutex.Unlock()
}

func GetEffectiveConfig() map[string]interface{} {
	return effectiveConfig
}
