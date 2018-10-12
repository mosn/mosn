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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	xdsboot "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/json-iterator/go"
)

type ContentKey string

// ServerConfig for making up server for mosn
type ServerConfig struct {
	//default logger
	ServerName      string `json:"mosn_server_name"`
	DefaultLogPath  string `json:"default_log_path,omitempty"`
	DefaultLogLevel string `json:"default_log_level,omitempty"`

	UseNetpollMode bool `json:"use_netpoll_mode,omitempty"`
	//graceful shutdown config
	GracefulTimeout v2.DurationConfig `json:"graceful_timeout"`

	//go processor number
	Processor int `json:"processor"`

	Listeners []v2.Listener `json:"listeners,omitempty"`
}

// ClusterManagerConfig for making up cluster manager
// Cluster is the global cluster of mosn
type ClusterManagerConfig struct {
	// Note: consider to use standard configure
	AutoDiscovery bool `json:"auto_discovery"`
	// Note: this is a hack method to realize cluster's  health check which push by registry
	RegistryUseHealthCheck bool         `json:"registry_use_health_check"`
	Clusters               []v2.Cluster `json:"clusters,omitempty"`
}

// MOSNConfig make up mosn to start the mosn project
// Servers contains the listener, filter and so on
// ClusterManager used to manage the upstream
type MOSNConfig struct {
	Servers         []ServerConfig         `json:"servers,omitempty"`         //server config
	ClusterManager  ClusterManagerConfig   `json:"cluster_manager,omitempty"` //cluster config
	ServiceRegistry v2.ServiceRegistryInfo `json:"service_registry"`          //service registry config, used by service discovery module
	//tracing config
	RawDynamicResources jsoniter.RawMessage `json:"dynamic_resources,omitempty"` //dynamic_resources raw message
	RawStaticResources  jsoniter.RawMessage `json:"static_resources,omitempty"`  //static_resources raw message
	RawAdmin            jsoniter.RawMessage `json:"admin,omitempty""`            // admin raw message
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

var (
	configPath string
	config     MOSNConfig
)

func (c *MOSNConfig) GetAdmin() *xdsboot.Admin {
	if len(c.RawAdmin) > 0 {
		adminConfig := &xdsboot.Admin{}
		err := jsonpb.UnmarshalString(string(config.RawAdmin), adminConfig)
		if err == nil {
			return adminConfig
		}
	}
	return nil
}

// Load config file and parse
func Load(path string) *MOSNConfig {
	log.Println("load config from : ", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln("load config failed, ", err)
		os.Exit(1)
	}
	configPath, _ = filepath.Abs(path)
	// translate to lower case
	err = json.Unmarshal(content, &config)
	if err != nil {
		log.Fatalln("json unmarshal config failed, ", err)
		os.Exit(1)
	}
	return &config
}
