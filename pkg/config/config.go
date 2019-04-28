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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/utils"
	"github.com/c2h5oh/datasize"
	xdsboot "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/gogo/protobuf/jsonpb"
)

type ContentKey string

// Tracing configuration for a server
type TracingConfig struct {
	Enable bool   `json:"enable"`
	Tracer string `json:"tracer"`
}

// MetricsConfig for metrics sinks
type MetricsConfig struct {
	SinkConfigs  []v2.Filter       `json:"sinks"`
	StatsMatcher v2.StatsMatcher   `json:"stats_matcher"`
	ShmZone      string            `json:"shm_zone"`
	ShmSize      datasize.ByteSize `json:"shm_size"`
}

// ClusterManagerConfig for making up cluster manager
// Cluster is the global cluster of mosn
type ClusterManagerConfig struct {
	ClusterManagerConfigJson
	Clusters []v2.Cluster `json:"-"`
}

type ClusterManagerConfigJson struct {
	// Note: consider to use standard configure
	AutoDiscovery bool `json:"auto_discovery,omitempty"`
	// Note: this is a hack method to realize cluster's  health check which push by registry
	RegistryUseHealthCheck bool         `json:"registry_use_health_check,omitempty"`
	ClusterConfigPath      string       `json:"clusters_configs,omitempty"`
	ClustersJson           []v2.Cluster `json:"clusters,omitempty"`
}

func (cc *ClusterManagerConfig) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &cc.ClusterManagerConfigJson); err != nil {
		return err
	}
	// only one of the config should be exists
	if len(cc.ClustersJson) > 0 && cc.ClusterConfigPath != "" {
		return v2.ErrDuplicateStaticAndDynamic
	}
	if len(cc.ClustersJson) > 0 {
		cc.Clusters = cc.ClustersJson
	}
	// Traversing path and parse the json
	// assume all the files in the path are available json file, and no sub path
	if cc.ClusterConfigPath != "" {
		files, err := ioutil.ReadDir(cc.ClusterConfigPath)
		if err != nil {
			return err
		}
		for _, f := range files {
			fileName := path.Join(cc.ClusterConfigPath, f.Name())
			data, err := ioutil.ReadFile(fileName)
			if err != nil {
				return err
			}
			cluster := v2.Cluster{}
			if err := json.Unmarshal(data, &cluster); err != nil {
				return err
			}
			cc.Clusters = append(cc.Clusters, cluster)
		}
	}
	return nil
}

// Marshal memory config into json, if dynamic mode is configured, write json file
func (cc ClusterManagerConfig) MarshalJSON() (b []byte, err error) {
	if cc.ClusterConfigPath == "" {
		cc.ClustersJson = cc.Clusters
		return json.Marshal(cc.ClusterManagerConfigJson)
	}
	// dynamic mode, should write file
	// first, get all the files in the directory
	files, err := ioutil.ReadDir(cc.ClusterConfigPath)
	if err != nil {
		return nil, err
	}
	allFiles := make(map[string]struct{}, len(files))
	for _, f := range files {
		allFiles[f.Name()] = struct{}{}
	}
	// file name is virtualhost name, if not exists, use {unixnano}.json
	for _, cluster := range cc.Clusters {
		fileName := cluster.Name
		if fileName == "" {
			fileName = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		data, err := json.MarshalIndent(cluster, "", " ")
		if err != nil {
			return nil, err
		}
		fileName = fileName + ".json"
		delete(allFiles, fileName)
		fileName = path.Join(cc.ClusterConfigPath, fileName)
		if err := utils.WriteFileSafety(fileName, data, 0644); err != nil {
			return nil, err
		}
	}
	// delete invalid files
	for f := range allFiles {
		os.Remove(path.Join(cc.ClusterConfigPath, f))
	}
	return json.Marshal(cc.ClusterManagerConfigJson)
}

// MOSNConfig make up mosn to start the mosn project
// Servers contains the listener, filter and so on
// ClusterManager used to manage the upstream
type MOSNConfig struct {
	Servers         []v2.ServerConfig      `json:"servers,omitempty"`         //server config
	ClusterManager  ClusterManagerConfig   `json:"cluster_manager,omitempty"` //cluster config
	ServiceRegistry v2.ServiceRegistryInfo `json:"service_registry"`          //service registry config, used by service discovery module
	//tracing config
	Tracing             TracingConfig   `json:"tracing"`
	Metrics             MetricsConfig   `json:"metrics"`
	RawDynamicResources json.RawMessage `json:"dynamic_resources,omitempty"` //dynamic_resources raw message
	RawStaticResources  json.RawMessage `json:"static_resources,omitempty"`  //static_resources raw message
	RawAdmin            json.RawMessage `json:"admin,omitempty"`             // admin raw message
	Debug               PProfConfig     `json:"pprof,omitempty"`
	Pid                 string          `json:"pid,omitempty"` // pid file
}

// PProfConfig is used to start a pprof server for debug
type PProfConfig struct {
	StartDebug bool `json:"debug"`      // If StartDebug is true, start a pprof, default is false
	Port       int  `json:"port_value"` // If port value is 0, will use 9090 as default
}

// Mode is mosn's starting type
type Mode uint8

// File means start from config file
// Xds means start from xds
// Mix means start both from file and Xds
const (
	File Mode = iota
	Xds
	Mix
)

func (c *MOSNConfig) Mode() Mode {
	if len(c.Servers) > 0 {
		if len(c.RawStaticResources) == 0 || len(c.RawDynamicResources) == 0 {
			return File
		}

		return Mix
	} else if len(c.RawStaticResources) > 0 && len(c.RawDynamicResources) > 0 {
		return Xds
	}

	return File
}

func (c *MOSNConfig) GetAdmin() *xdsboot.Admin {
	if len(c.RawAdmin) > 0 {
		adminConfig := &xdsboot.Admin{}
		err := jsonpb.UnmarshalString(string(c.RawAdmin), adminConfig)
		if err == nil {
			return adminConfig
		}
	}
	return nil
}

// protetced configPath, read only
func GetConfigPath() string {
	return configPath
}
