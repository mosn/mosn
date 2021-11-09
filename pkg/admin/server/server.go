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

package server

import (
	"fmt"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"mosn.io/mosn/pkg/admin/store"
	"mosn.io/mosn/pkg/log"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// apiHandlerStore stores the supported admin api
// can register more admin api
var apiHandlerStore map[string]*APIHandler

// RegisterAdminHandleFunc keeps compatible for old ways
func RegisterAdminHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	apiHandlerStore[pattern] = NewAPIHandler(handler, nil)
	log.StartLogger.Infof("[admin server] [register api] register a new api %s", pattern)
}

// RegisterAdminHandler registers a API Handler for amdin api, the Handler can contains auths and failed action.
func RegisterAdminHandler(pattern string, handler *APIHandler) {
	apiHandlerStore[pattern] = handler
	log.StartLogger.Infof("[admin server] [register api] register a new api %s", pattern)
}

// DeleteRegisteredAdminHandler deletes a registered pattern
func DeleteRegisteredAdminHandler(pattern string) {
	delete(apiHandlerStore, pattern)
	log.StartLogger.Infof("[admin server] [register api] delete registered api %s", pattern)
}

func init() {
	// default admin api
	apiHandlerStore = map[string]*APIHandler{
		"/api/v1/config_dump":     NewAPIHandler(ConfigDump, nil),
		"/api/v1/stats":           NewAPIHandler(StatsDump, nil),
		"/api/v1/stats_glob":      NewAPIHandler(StatsDumpProxyTotal, nil),
		"/api/v1/update_loglevel": NewAPIHandler(UpdateLogLevel, nil),
		"/api/v1/get_loglevel":    NewAPIHandler(GetLoggerInfo, nil),
		"/api/v1/enable_log":      NewAPIHandler(EnableLogger, nil),
		"/api/v1/disable_log":     NewAPIHandler(DisableLogger, nil),
		"/api/v1/states":          NewAPIHandler(GetState, nil),
		"/api/v1/plugin":          NewAPIHandler(PluginApi, nil),
		"/api/v1/features":        NewAPIHandler(KnownFeatures, nil),
		"/api/v1/env":             NewAPIHandler(GetEnv, nil),
		"/":                       NewAPIHandler(Help, nil),
	}
}

type Server struct {
	*http.Server
}

func (s *Server) Start(config Config) {
	var addr string
	if config != nil {
		// get admin config
		adminConfig := config.GetAdmin()
		if adminConfig == nil {
			// no admin config, no admin start
			log.DefaultLogger.Warnf("no admin config, no admin api served")
			return
		}
		addr = fmt.Sprintf("%s:%d", adminConfig.GetAddress(), adminConfig.GetPortValue())
	}

	mux := http.NewServeMux()
	for pattern, handler := range apiHandlerStore {
		mux.Handle(pattern, handler)
	}

	srv := &http.Server{Addr: addr, Handler: mux}
	store.AddService(srv, "Mosn Admin Server", nil, nil)
	s.Server = srv
}
