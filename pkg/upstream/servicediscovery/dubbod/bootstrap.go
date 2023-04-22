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
package dubbod

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/go-chi/chi"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	routerAdapter "mosn.io/mosn/pkg/router"
	dubbologger "mosn.io/pkg/registry/dubbo/common/logger"
	"mosn.io/pkg/utils"
)

// dubboConfig is for dubbo related configs
type dubboConfig struct {
	Enable             bool   `json:"enable"`
	ServerListenerPort int    `json:"server_port"` // for server listener, must keep the same with server listener
	APIPort            int    `json:"api_port"`    // for pub/sub/unpub/unsub api
	LogPath            string `json:"log_path"`    // dubbo service discovery log_path
}

/*
example config :

	   "extends": {
		   "dubbo_registry": {
			    "enable" : true,
			    "server_port" : 20080,
			    "api_port" : 10022,
			    "log_path" : "/tmp"
		   }
	   }
*/
func init() {
	v2.RegisterParseExtendConfig("dubbo_registry", func(config json.RawMessage) error {
		var conf dubboConfig
		err := json.Unmarshal(config, &conf)
		if err != nil {
			log.DefaultLogger.Errorf("[dubbod] failed to parse dubbo registry config: %v", err.Error())
			return err
		}
		if conf.Enable {
			Init(conf.APIPort, conf.ServerListenerPort, conf.LogPath)
		}
		return nil
	})
}

// Init set the pub port && api port
// and init dubbo API service
func Init(configAPIPort, configPubPort int, logPath string) {
	initGlobalVars(configAPIPort, configPubPort, logPath)
	initAPI()
}

func initGlobalVars(configAPIPort, configPubPort int, logFullPath string) {
	if configAPIPort > 0 && configAPIPort < 65535 {
		// valid port
		apiPort = fmt.Sprint(configAPIPort)
	} else {
		log.DefaultLogger.Warnf("invalid dubbo api port: %v, will use default: %v", configAPIPort, apiPort)
	}

	if configPubPort > 0 && configPubPort < 65535 {
		// valid port
		mosnPubPort = fmt.Sprint(configPubPort)
	} else {
		log.DefaultLogger.Fatalf("invalid dubbo pub port: %v, will use default: %v", configPubPort, mosnPubPort)
	}

	if len(logFullPath) > 0 {
		logPath = logFullPath
	}
}

var apiPort = "22222" // default 22222
var logPath = "./dubbogo.log"

// initAPI inits the http api for application when application bootstrap
// for sub/pub
func initAPI() {
	// 1. init router
	r := chi.NewRouter()
	r.Post("/sub", subscribe)
	r.Post("/unsub", unsubscribe)
	r.Post("/pub", publish)
	r.Post("/unpub", unpublish)

	_ = dubbologger.InitLog(path.Join(logPath, "dubbo.log"))

	// FIXME make port configurable
	utils.GoWithRecover(func() {
		if err := http.ListenAndServe(":"+apiPort, r); err != nil {
			log.DefaultLogger.Infof("auto write config when updated")
		}
	}, nil)

	// 2. init dubbo router
	initRouterManager()
}

// inject a router to router manager
// which name is "dubbo"
func initRouterManager() {
	// this can also be put into mosn's config json file
	err := routerAdapter.GetRoutersMangerInstance().AddOrUpdateRouters(&v2.RouterConfiguration{
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: dubboRouterConfigName,
		},
		VirtualHosts: []v2.VirtualHost{
			{
				Name:    dubboRouterConfigName,
				Domains: []string{"*"},
			},
		},
	})
	if err != nil {
		log.DefaultLogger.Fatalf("auto write config when updated")
	}
}
