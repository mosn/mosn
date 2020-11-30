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
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/ghodss/yaml"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

var (
	// configPath stores the config file path
	configPath string
	// dyconfigPath stores another config file path in which only cluster,listener,router will be loaded
	dyconfigPath string
	// configLock controls the stored config
	configLock sync.RWMutex
	// conf keeps the mosn config
	conf effectiveConfig
	// configLoadFunc can be replaced by load config extension
	configLoadFunc ConfigLoadFunc = DefaultConfigLoad
)

// protetced configPath, read only
func GetConfigPath() string {
	return configPath
}

// ConfigLoadFunc parse a input(usually file path) into a mosn config
type ConfigLoadFunc func(path string) *v2.MOSNConfig

// RegisterConfigLoadFunc can replace a new config load function instead of default
func RegisterConfigLoadFunc(f ConfigLoadFunc) {
	configLoadFunc = f
}

func DefaultConfigLoad(path string) *v2.MOSNConfig {
	log.StartLogger.Infof("load config from :  %s", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.StartLogger.Fatalf("[config] [default load] load config failed, error: %v", err)
	}
	cfg := &v2.MOSNConfig{}
	if yamlFormat(path) {
		bytes, err := yaml.YAMLToJSON(content)
		if err != nil {
			log.StartLogger.Fatalf("[config] [default load] translate yaml to json error: %v", err)
		}
		content = bytes
	}
	// translate to lower case
	err = json.Unmarshal(content, cfg)
	if err != nil {
		log.StartLogger.Fatalf("[config] [default load] json unmarshal config failed, error: %v", err)
	}
	return cfg

}

// Load config file and parse
func Load(path string) *v2.MOSNConfig {
	configPath, _ = filepath.Abs(path)
	cfg := configLoadFunc(path)
	if dyconfigPath != "" {
		checkUpdateFromDyconfig(cfg)
	}
	return cfg
}

func yamlFormat(path string) bool {
	ext := filepath.Ext(path)
	if ext == ".yaml" || ext == ".yml" {
		return true
	}
	return false
}

func checkUpdateFromDyconfig(conf *v2.MOSNConfig) {
	if _, err := os.Stat(dyconfigPath); os.IsNotExist(err) {
		return
	}
	if cfg := configLoadFunc(dyconfigPath); cfg != nil {
		// update clusters
		l := len(conf.ClusterManager.Clusters)
		for _, cluster := range cfg.ClusterManager.Clusters {
			found := false
			for i := 0; i < l; i++ {
				if cluster.Name == conf.ClusterManager.Clusters[i].Name {
					log.StartLogger.Debugf("[config] [load dyconfig] cluster %s added\n", cluster.Name)
					conf.ClusterManager.Clusters[i] = cluster
					found = true
					break
				}
			}
			if !found {
				log.StartLogger.Debugf("[config] [load dyconfig] cluster %s added\n", cluster.Name)
				conf.ClusterManager.Clusters = append(conf.ClusterManager.Clusters, cluster)
			}
		}
		// server config should not be empty in dynamic config
		if len(cfg.Servers) == 0 {
			log.StartLogger.Fatalf("[config] no server found in config file %s", dyconfigPath)
		}
		if len(conf.Servers) > 0 {
			// update listeners
			for _, listener := range cfg.Servers[0].Listeners {
				found := false
				for i := 0; i < len(conf.Servers[0].Listeners); i++ {
					if listener.Name == conf.Servers[0].Listeners[i].Name {
						log.StartLogger.Debugf("[config] [load dyconfig] lisntener %s updated\n", listener.Name)
						conf.Servers[0].Listeners[i] = listener
						found = true
						break
					}
				}
				if !found {
					log.StartLogger.Debugf("[config] [load dyconfig] listener %s added\n", listener.Name)
					conf.Servers[0].Listeners = append(conf.Servers[0].Listeners, listener)
				}
			}
			// update routers
			for _, router := range cfg.Servers[0].Routers {
				found := false
				for i := 0; i < len(conf.Servers[0].Routers); i++ {
					if router.RouterConfigName == conf.Servers[0].Routers[i].RouterConfigName {
						log.StartLogger.Debugf("[config] [load dyconfig] router %s updated\n", router.RouterConfigName)
						conf.Servers[0].Routers[i] = router
						found = true
						break
					}
				}
				if !found {
					log.StartLogger.Infof("[config] [load dyconfig] router %s added\n", router.RouterConfigName)
					conf.Servers[0].Routers = append(conf.Servers[0].Routers, router)
				}
			}
		} else {
			conf.Servers = append(conf.Servers, cfg.Servers[0])
		}
	}
}

// Load config file and parse
func SetDynamicConfigPath(path string) {
	dyconfigPath, _ = filepath.Abs(path)
}
