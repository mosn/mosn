package server

import (
	"bytes"
	"fmt"
	"net/http"

	envoyControlPlaneAPI "github.com/envoyproxy/go-control-plane/envoy/admin/v2alpha"
	"github.com/golang/protobuf/jsonpb"
	"mosn.io/mosn/pkg/admin/store"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/xds/conv"
)

const (
	CDS_UPDATE_SUCCESS   = "cluster_manager.cds.update_success"
	CDS_UPDATE_REJECT    = "cluster_manager.cds.update_rejected"
	LDS_UPDATE_SUCCESS   = "listener_manager.lds.update_success"
	LDS_UPDATE_REJECT    = "listener_manager.lds.update_rejected"
	SERVER_STATE         = "server.state"
	STAT_WORKERS_STARTED = "listener_manager.workers_started"
)

func statsForIstio(w http.ResponseWriter, r *http.Request) {
	state, err := getIstioState()
	if err != nil {
		log.DefaultLogger.Warnf("%v", err)
	}
	var workersStarted int
	if state == envoyControlPlaneAPI.ServerInfo_LIVE {
		workersStarted = 1
	}

	sb := bytes.NewBufferString("")
	sb.WriteString(fmt.Sprintf("%s: %d\n", CDS_UPDATE_SUCCESS, conv.Stats.CdsUpdateSuccess.Count()))
	sb.WriteString(fmt.Sprintf("%s: %d\n", CDS_UPDATE_REJECT, conv.Stats.CdsUpdateReject.Count()))
	sb.WriteString(fmt.Sprintf("%s: %d\n", LDS_UPDATE_SUCCESS, conv.Stats.LdsUpdateSuccess.Count()))
	sb.WriteString(fmt.Sprintf("%s: %d\n", LDS_UPDATE_REJECT, conv.Stats.LdsUpdateReject.Count()))
	sb.WriteString(fmt.Sprintf("%s: %d\n", SERVER_STATE, state))
	sb.WriteString(fmt.Sprintf("%s: %d\n", STAT_WORKERS_STARTED, workersStarted))
	_, err = sb.WriteTo(w)

	if err != nil {
		log.DefaultLogger.Warnf("%v", err)
	}
}

func serverInfoForIstio(w http.ResponseWriter, r *http.Request) {
	i := envoyControlPlaneAPI.ServerInfo{}
	var err error
	i.State, err = getIstioState()
	if err != nil {
		log.DefaultLogger.Warnf("%v", err)
		return
	}

	m := jsonpb.Marshaler{}
	if err := m.Marshal(w, &i); err != nil {
		log.DefaultLogger.Warnf("marshal to string failed")
	}
}

func getIstioState() (envoyControlPlaneAPI.ServerInfo_State, error) {
	mosnState := store.GetMosnState()

	switch mosnState {
	case store.Active_Reconfiguring:
		return envoyControlPlaneAPI.ServerInfo_PRE_INITIALIZING, nil
	case store.Init:
		return envoyControlPlaneAPI.ServerInfo_INITIALIZING, nil
	case store.Running:
		return envoyControlPlaneAPI.ServerInfo_LIVE, nil
	case store.Passive_Reconfiguring:
		return envoyControlPlaneAPI.ServerInfo_DRAINING, nil
	}

	return 0, fmt.Errorf("parse mosn state %v to istio state failed", mosnState)
}
