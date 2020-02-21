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
	"sync/atomic"
	"time"

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/featuregate"
	"mosn.io/pkg/log"
	"mosn.io/pkg/utils"
)

// ConfigAutoFeature controls xDS update config will overwrite config file or not, default is not
type ConfigAutoFeature struct {
	featuregate.BaseFeatureSpec
	dump uint32
}

const ConfigAutoWrite featuregate.Feature = "auto_config"

var feature *ConfigAutoFeature

func tryDump() {
	feature.dumpConfig()
}

func init() {
	feature = &ConfigAutoFeature{
		BaseFeatureSpec: featuregate.BaseFeatureSpec{
			DefaultValue: false,
		},
		dump: 0,
	}
	featuregate.AddFeatureSpec(ConfigAutoWrite, feature)
}

func (f *ConfigAutoFeature) dumpConfig() {
	if !f.State() {
		return
	}
	atomic.CompareAndSwapUint32(&f.dump, 0, 1)
}

func (f *ConfigAutoFeature) doDumpConfig() {
	if atomic.CompareAndSwapUint32(&f.dump, 1, 0) {
		mutex.Lock()
		defer mutex.Unlock()
		var listeners []v2.Listener
		var routers []*v2.RouterConfiguration
		var clusters []v2.Cluster
		log.DefaultLogger.Debugf("try to dump full config")
		// get listeners
		for _, l := range conf.Listener {
			listeners = append(listeners, l)
		}
		// get clusters
		for _, c := range conf.Cluster {
			clusters = append(clusters, c)
		}
		// get routers, should set the original path
		for name := range conf.Routers {
			routerPath := conf.routerConfigPath[name]
			r := conf.Routers[name] // copy the value
			r.RouterConfigPath = routerPath
			routers = append(routers, &r)
		}
		configmanager.UpdateFullConfig(listeners, routers, clusters)
	}
}

func (f *ConfigAutoFeature) InitFunc() {
	utils.GoWithRecover(func() {
		log.DefaultLogger.Infof("auto write config when updated")
		for {
			f.doDumpConfig()
			time.Sleep(3 * time.Second)
		}
	}, nil)
}
