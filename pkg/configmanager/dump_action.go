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
	"sync"
	"sync/atomic"
	"time"

	"github.com/ghodss/yaml"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

var (
	// dumping represents the dump state
	dumping int32
	once    sync.Once
	// we call dump action in a lock, in case file dump when smooth upgrade
	lock sync.Mutex
)

func DumpLock() {
	lock.Lock()
}

func DumpUnlock() {
	lock.Unlock()
}

// DumpConfigHandler should be called in a goroutine
// we call it in mosn/starter with GoWithRecover, which can handle the panic information
func DumpConfigHandler() {
	once.Do(func() {
		for {
			time.Sleep(3 * time.Second)
			DumpLock()
			DumpConfig()
			DumpUnlock()
		}
	})
}

// setDump marks the dumping state and trigger dump at next dump check loop
func setDump() {
	atomic.CompareAndSwapInt32(&dumping, 0, 1)
}

// getDump taks the dumping state, if true, do dump action
func getDump() bool {
	return atomic.CompareAndSwapInt32(&dumping, 1, 0)
}

// DumpConfig writes the config into config path file
func DumpConfig() {
	if !getDump() {
		return
	}
	log.DefaultLogger.Debugf("[config] [dump] try to dump config")
	content, err := transferConfig()
	if err != nil {
		log.DefaultLogger.Alertf(types.ErrorKeyConfigDump, "dump config failed, caused by: %v", err)
		return
	}
	// check if yaml
	if yamlFormat(configPath) {
		content, err = yaml.JSONToYAML(content)
		if err != nil {
			log.DefaultLogger.Alertf(types.ErrorKeyConfigDump, "dump yaml config failed, caused by: %v", err)
			return
		}
	}
	err = utils.WriteFileSafety(configPath, content, 0644)
	if err != nil {
		log.DefaultLogger.Alertf(types.ErrorKeyConfigDump, "dump config failed, caused by: %v", err.Error())
		// add retry if write file  failed
		setDump()
	}
}

// transferExtensionFunc is an extension for handle config before dump
// if auto_config is enabled, this function never be called
var transferExtensionFunc func(*v2.MOSNConfig)

// RegisterTransferExtension registers the transferExtensionFunc.
// this function should be called before dump goroutine is started.
func RegisterTransferExtension(f func(*v2.MOSNConfig)) {
	transferExtensionFunc = f
}

// transferConfig makes effective config to v2.MOSNConfig bytes
func transferConfig() ([]byte, error) {
	configLock.RLock()
	defer configLock.RUnlock()

	wait2dump := conf.MosnConfig
	listeners := make([]v2.Listener, 0, len(conf.Listener))
	for _, l := range conf.Listener {
		listeners = append(listeners, l)
	}
	clusters := make([]v2.Cluster, 0, len(conf.Cluster))
	for _, c := range conf.Cluster {
		clusters = append(clusters, c)
	}
	routers := make([]*v2.RouterConfiguration, 0, len(conf.Routers))
	// get routers, should set the original path
	for name := range conf.Routers {
		routerPath := conf.routerConfigPath[name]
		r := conf.Routers[name] // copy the value
		r.RouterConfigPath = routerPath
		routers = append(routers, &r)
	}
	extends := make([]v2.ExtendConfig, len(conf.ExtendConfigs))
	for k, v := range conf.ExtendConfigs {
		extends[k] = v
	}
	if len(wait2dump.Servers) == 0 {
		log.DefaultLogger.Errorf("no server config should only exists in test mode. mock an empty one")
		wait2dump.Servers = make([]v2.ServerConfig, 1)
		wait2dump.Servers[0] = v2.ServerConfig{}
	}
	wait2dump.Servers[0].Listeners = listeners
	wait2dump.Servers[0].Routers = routers
	wait2dump.ClusterManager.Clusters = clusters
	wait2dump.ClusterManager.ClusterConfigPath = conf.clusterConfigPath
	wait2dump.Extends = extends
	if transferExtensionFunc != nil {
		transferExtensionFunc(&wait2dump)
	}
	return json.MarshalIndent(wait2dump, "", "  ")
}

// InheritMosnconfig get old mosn configs
func InheritMosnconfig() ([]byte, error) {
	return transferConfig()
}
