//go:build mosn_debug
// +build mosn_debug

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

package debug

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	admin "mosn.io/mosn/pkg/admin/server"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/server"
	"mosn.io/mosn/pkg/upstream/cluster"
)

// This pakcage is only builded in debug mode/test mode.
// Production usage should not build with tags `mosn_debug`.
// The functions in this package is dangerous in production.
type UpdateConfigRequest struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

func init() {
	log.StartLogger.Infof("mosn is builded in debug mosn")
	admin.RegisterAdminHandleFunc("/debug/update_config", DebugUpdateMosnConfig)
	admin.RegisterAdminHandleFunc("/debug/disable_tls", DebugUpdateTLSDisable)
	admin.RegisterAdminHandleFunc("/debug/update_route", DebugUdpateRoute)
}

// The config types support to be updated
const (
	typListener = "listener"
	typRouter   = "router"
	typCluster  = "cluster" // include hosts
	typeExtend  = "extend"
)

var success = []byte("success")

func DebugUpdateMosnConfig(w http.ResponseWriter, r *http.Request) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.DefaultLogger.Errorf("api [update mosn config] read body error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "invalid request")
		return
	}
	invalid := func(s string) {
		log.DefaultLogger.Errorf("api [update mosn config] is not a valid request: %s", s)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "invalid request")

	}
	req := &UpdateConfigRequest{}
	if err := json.Unmarshal(content, req); err != nil {
		invalid(string(content))
		return
	}
	switch req.Type {
	case typListener:
		ln := &v2.Listener{}
		if err := json.Unmarshal(req.Config, ln); err != nil {
			invalid(string(req.Config))
			return
		}
		adapter := server.GetListenerAdapterInstance()
		// we support only one server now. the server name is equal to default server name
		// use empty server name to index default server
		if err := adapter.AddOrUpdateListener("", ln); err != nil {
			invalid(err.Error())
			return
		}
		log.DefaultLogger.Infof("update listener config success")
		w.Write(success)
	case typRouter:
		r := &v2.RouterConfiguration{}
		if err := json.Unmarshal(req.Config, r); err != nil {
			invalid(string(req.Config))
			return
		}
		mng := router.GetRoutersMangerInstance()
		if err := mng.AddOrUpdateRouters(r); err != nil {
			invalid(err.Error())
			return
		}
		log.DefaultLogger.Infof("update router config success")
		w.Write(success)
	case typCluster:
		c := v2.Cluster{}
		if err := json.Unmarshal(req.Config, &c); err != nil {
			invalid(string(req.Config))
			return
		}
		adapter := cluster.GetClusterMngAdapterInstance()
		if err := adapter.TriggerClusterAndHostsAddOrUpdate(c, c.Hosts); err != nil {
			invalid(err.Error())
			return
		}
		log.DefaultLogger.Infof("update cluster config success")
		w.Write(success)
	case typeExtend:
		ext := []v2.ExtendConfig{}
		if err := json.Unmarshal(req.Config, &ext); err != nil {
			invalid(string(req.Config))
			return
		}
		// just update config
		for _, c := range ext {
			configmanager.SetExtend(c.Type, c.Config)
		}
		log.DefaultLogger.Infof("update extend config success")
		w.Write(success)
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "invalid type, do nothing")
	}
}

func DebugUpdateTLSDisable(w http.ResponseWriter, r *http.Request) {
	invalid := func(s string) {
		log.DefaultLogger.Errorf("api [update mosn config] is not a valid request: %s", s)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "invalid request")
	}
	r.ParseForm()
	v := r.FormValue("disable")
	t, err := strconv.ParseBool(v)
	if err != nil {
		invalid(err.Error())
		return
	}
	if t {
		log.DefaultLogger.Infof("disable global tls")
		cluster.DisableClientSideTLS()
	} else {
		log.DefaultLogger.Infof("enable global tls")
		cluster.EnableClientSideTLS()
	}
	w.Write(success)
}

const (
	AddRoute    = "add"
	RemoveRoute = "remove"
)

type RouteConfig struct {
	RouteName string     `json:"router_config_name"`
	Domain    string     `json:"domain"`
	Route     *v2.Router `json:"route"`
}

func DebugUdpateRoute(w http.ResponseWriter, r *http.Request) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.DefaultLogger.Errorf("api [update mosn config] read body error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "invalid request")
		return
	}
	invalid := func(s string) {
		log.DefaultLogger.Errorf("api [update mosn config] is not a valid request: %s", s)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "invalid request")
	}
	req := &UpdateConfigRequest{}
	if err := json.Unmarshal(content, req); err != nil {
		invalid(string(content))
		return
	}
	switch req.Type {
	case AddRoute:
		rcfg := &RouteConfig{}
		if err := json.Unmarshal(req.Config, rcfg); err != nil {
			invalid(string(req.Config))
			return
		}
		mng := router.GetRoutersMangerInstance()
		if err := mng.AddRoute(rcfg.RouteName, rcfg.Domain, rcfg.Route); err != nil {
			invalid(err.Error())
			return
		}
		log.DefaultLogger.Infof("update route config success")
		w.Write(success)
	case RemoveRoute:
		rcfg := &RouteConfig{}
		if err := json.Unmarshal(req.Config, rcfg); err != nil {
			invalid(string(req.Config))
			return
		}
		mng := router.GetRoutersMangerInstance()
		if err := mng.RemoveAllRoutes(rcfg.RouteName, rcfg.Domain); err != nil {
			invalid(err.Error())
			return
		}
		log.DefaultLogger.Infof("remove all route success")
		w.Write(success)
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "invalid type, do nothing")
	}

}
