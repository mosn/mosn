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
	"net/http"

	v2 "mosn.io/mosn/pkg/config/v2"
	routerAdapter "mosn.io/mosn/pkg/router"

	"github.com/go-chi/chi"
	dubbologger "github.com/mosn/registry/dubbo/common/logger"
	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/utils"
)

// init the http api for application when application bootstrap
// for sub/pub
func Init( /*port string, dubboLogPath string*/ ) {
	// 1. init router
	r := chi.NewRouter()
	r.Post("/sub", subscribe)
	r.Post("/unsub", unsubscribe)
	r.Post("/pub", publish)
	r.Post("/unpub", unpublish)

	_ = dubbologger.InitLog("./dubbogo.log")

	// FIXME make port configurable
	utils.GoWithRecover(func() {
		if err := http.ListenAndServe(":22222", r); err != nil {
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
		VirtualHosts: []*v2.VirtualHost{
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
