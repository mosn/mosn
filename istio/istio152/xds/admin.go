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

	envoy_admin_v2alpha "github.com/envoyproxy/go-control-plane/envoy/admin/v2alpha"
	"github.com/golang/protobuf/jsonpb"
	"mosn.io/mosn/pkg/admin/server"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/stagemanager"
)

const (
	CDS_UPDATE_SUCCESS   = "cluster_manager.cds.update_success"
	CDS_UPDATE_REJECT    = "cluster_manager.cds.update_rejected"
	LDS_UPDATE_SUCCESS   = "listener_manager.lds.update_success"
	LDS_UPDATE_REJECT    = "listener_manager.lds.update_rejected"
	SERVER_STATE         = "server.state"
	STAT_WORKERS_STARTED = "listener_manager.workers_started"
)

func init() {
	server.RegisterAdminHandleFunc("/server_info", serverInfoForIstio)
}

func (ads *AdsConfig) statsForIstio(w http.ResponseWriter, _ *http.Request) {
	state, err := getIstioState()
	if err != nil {
		log.DefaultLogger.Warnf("get istio state error : %v", err)
	}
	var workersStarted int
	if state == envoy_admin_v2alpha.ServerInfo_LIVE {
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

func serverInfoForIstio(w http.ResponseWriter, _ *http.Request) {
	i := envoy_admin_v2alpha.ServerInfo{}
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

func getIstioState() (envoy_admin_v2alpha.ServerInfo_State, error) {
	state := stagemanager.GetState()
	switch state {
	case stagemanager.Nil:
		fallthrough
	case stagemanager.ParamsParsed:
		fallthrough
	case stagemanager.Initing:
		return envoy_admin_v2alpha.ServerInfo_INITIALIZING, nil

	case stagemanager.PreStart:
		fallthrough
	case stagemanager.Starting:
		return envoy_admin_v2alpha.ServerInfo_PRE_INITIALIZING, nil

	case stagemanager.AfterStart:
		fallthrough
	case stagemanager.StartingNewServer: // new server not started yet.
		fallthrough
	case stagemanager.Running:
		return envoy_admin_v2alpha.ServerInfo_LIVE, nil

	case stagemanager.PreStop:
		fallthrough
	case stagemanager.Stopping:
		fallthrough
	case stagemanager.AfterStop:
		fallthrough
	case stagemanager.Upgrading:
		return envoy_admin_v2alpha.ServerInfo_DRAINING, nil
	}

	return 0, fmt.Errorf("parse mosn state %v to istio state failed", state)
}
