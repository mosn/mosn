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

package xds

import (
	"bytes"
	"fmt"
	"net/http"

	envoyControlPlaneAPI "github.com/envoyproxy/go-control-plane/envoy/admin/v3"
	"github.com/golang/protobuf/jsonpb"
	"mosn.io/mosn/istio/istio1106/xds/conv"
	"mosn.io/mosn/pkg/admin/server"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/stagemanager"
	"mosn.io/mosn/pkg/types"
)

const (
	CDS_UPDATE_SUCCESS   = "cluster_manager.cds.update_success"
	CDS_UPDATE_REJECT    = "cluster_manager.cds.update_rejected"
	LDS_UPDATE_SUCCESS   = "listener_manager.lds.update_success"
	LDS_UPDATE_REJECT    = "listener_manager.lds.update_rejected"
	SERVER_STATE         = "server.state"
	STAT_WORKERS_STARTED = "listener_manager.workers_started"
)

var (
	mosnState2IstioState = map[stagemanager.State]envoyControlPlaneAPI.ServerInfo_State{
		stagemanager.Nil: envoyControlPlaneAPI.ServerInfo_PRE_INITIALIZING,
		// 10 main stages
		stagemanager.ParamsParsed:     envoyControlPlaneAPI.ServerInfo_PRE_INITIALIZING,
		stagemanager.Initing:          envoyControlPlaneAPI.ServerInfo_PRE_INITIALIZING,
		stagemanager.PreStart:         envoyControlPlaneAPI.ServerInfo_INITIALIZING,
		stagemanager.Starting:         envoyControlPlaneAPI.ServerInfo_INITIALIZING,
		stagemanager.AfterStart:       envoyControlPlaneAPI.ServerInfo_LIVE,
		stagemanager.Running:          envoyControlPlaneAPI.ServerInfo_LIVE,
		stagemanager.GracefulStopping: envoyControlPlaneAPI.ServerInfo_DRAINING,
		stagemanager.Stopping:         envoyControlPlaneAPI.ServerInfo_DRAINING,
		stagemanager.AfterStop:        envoyControlPlaneAPI.ServerInfo_DRAINING,
		stagemanager.Stopped:          envoyControlPlaneAPI.ServerInfo_DRAINING,
		// 2 additional stages
		stagemanager.StartingNewServer: envoyControlPlaneAPI.ServerInfo_LIVE,
		stagemanager.Upgrading:         envoyControlPlaneAPI.ServerInfo_DRAINING,
	}
)

func init() {
	server.RegisterAdminHandleFunc("/server_info", serverInfoForIstio)
	server.RegisterAdminHandleFunc("/config_dump", envoyConfigDump)
}

func (ads *AdsConfig) statsForIstio(w http.ResponseWriter, _ *http.Request) {
	state, err := getIstioState()
	if err != nil {
		log.DefaultLogger.Warnf("get istio state error : %v", err)
	}
	var workersStarted int
	if state == envoyControlPlaneAPI.ServerInfo_LIVE {
		workersStarted = 1
	}

	sb := bytes.NewBufferString("")
	stats := ads.converter.Stats()
	sb.WriteString(fmt.Sprintf("%s: %d\n", CDS_UPDATE_SUCCESS, stats.CdsUpdateSuccess.Count()))
	sb.WriteString(fmt.Sprintf("%s: %d\n", CDS_UPDATE_REJECT, stats.CdsUpdateReject.Count()))
	sb.WriteString(fmt.Sprintf("%s: %d\n", LDS_UPDATE_SUCCESS, stats.LdsUpdateSuccess.Count()))
	sb.WriteString(fmt.Sprintf("%s: %d\n", LDS_UPDATE_REJECT, stats.LdsUpdateReject.Count()))
	sb.WriteString(fmt.Sprintf("%s: %d\n", SERVER_STATE, state))
	sb.WriteString(fmt.Sprintf("%s: %d\n", STAT_WORKERS_STARTED, workersStarted))
	_, err = sb.WriteTo(w)

	if err != nil {
		log.DefaultLogger.Warnf("write stats for istio response error: %v", err)
	}

}

func serverInfoForIstio(w http.ResponseWriter, r *http.Request) {
	i := envoyControlPlaneAPI.ServerInfo{}
	var err error
	i.State, err = getIstioState()
	if err != nil {
		log.DefaultLogger.Warnf("get server info for istio state error : %v", err)
		return
	}

	m := jsonpb.Marshaler{}
	if err := m.Marshal(w, &i); err != nil {
		log.DefaultLogger.Warnf("get server info for istio marshal to string failed: %v", err)
	}
}

func getIstioState() (envoyControlPlaneAPI.ServerInfo_State, error) {
	state := stagemanager.GetState()
	if s, ok := mosnState2IstioState[state]; ok {
		return s, nil
	}

	return 0, fmt.Errorf("parse mosn state %v to istio state failed", state)
}

func envoyConfigDump(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, error: invalid method: %s", "config dump envoy", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	if len(r.Form) == 0 {
		buf := conv.EnvoyConfigDump()
		log.DefaultLogger.Infof("[admin api] [config dump envoy] config dump envoy")
		w.WriteHeader(200)
		w.Write(buf)
		return
	}
	if len(r.Form) > 1 {
		w.WriteHeader(400)
		fmt.Fprintf(w, "only support one parameter")
		return
	}
	log.DefaultLogger.Warnf("[admin api] [config dump envoy] r.Form %+v", r.Form)
}
