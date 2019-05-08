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
	"encoding/json"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alipay/sofa-mosn/pkg/admin/store"
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/utils"
)

var (
	once    sync.Once
	lock    sync.Mutex
	dumping int32
)

func DumpLock() {
	lock.Lock()
}

func DumpUnlock() {
	lock.Unlock()
}

func setDump() {
	atomic.CompareAndSwapInt32(&dumping, 0, 1)
}

func getDump() bool {
	return atomic.CompareAndSwapInt32(&dumping, 1, 0)
}

type routerConfigMap struct {
	config map[string]*v2.RouterConfiguration
	sync.Mutex
}

var routerMap = &routerConfigMap{
	config: make(map[string]*v2.RouterConfiguration),
}

func dumpRouterConfig() bool {
	routerMap.Lock()
	defer routerMap.Unlock()
	for listenername, routerConfig := range routerMap.config {
		ln, idx := findListener(listenername)
		if idx == -1 {
			continue
		}
		delete(routerMap.config, listenername)
		// support only one filter chain
		nfs := ln.FilterChains[0].Filters
		filterIndex := -1
		for i, nf := range nfs {
			if nf.Type == v2.CONNECTION_MANAGER {
				filterIndex = i
				break
			}
		}

		if data, err := json.MarshalIndent(routerConfig, "", " "); err == nil {
			cfg := make(map[string]interface{})
			if err := json.Unmarshal(data, &cfg); err != nil {
				log.DefaultLogger.Errorf("invalid router config, update config failed")
				continue
			}
			filter := v2.Filter{
				Type:   v2.CONNECTION_MANAGER,
				Config: cfg,
			}
			if filterIndex == -1 {
				nfs = append(nfs, filter)
				ln.FilterChains[0].Filters = nfs
				updateListener(idx, ln)
			} else {
				nfs[filterIndex] = filter
			}
		}
	}
	return true
}

func dump(dirty bool) {
	if dirty {
		setDump()
	}
}

func DumpConfig() {
	if getDump() {
		//update router config
		dumpRouterConfig()

		log.DefaultLogger.Debugf("dump config content: %+v", config)

		//update mosn_config
		store.SetMOSNConfig(config)
		// use golang original json lib, so the marshal ident can handle MarshalJSON interface implement correctly
		content, err := json.MarshalIndent(config, "", "  ")
		if err == nil {
			err = utils.WriteFileSafety(configPath, content, 0644)
		}

		if err != nil {
			log.DefaultLogger.Errorf("dump config failed, caused by: " + err.Error())
		}
	}
}

func DumpConfigHandler() {
	once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				log.DefaultLogger.Errorf("panic %v\n%s", r, string(debug.Stack()))
			}
		}()

		for {
			time.Sleep(3 * time.Second)

			DumpLock()
			DumpConfig()
			DumpUnlock()
		}
	})
}
