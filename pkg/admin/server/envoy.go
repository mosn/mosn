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
	CDS_UPDATE_SUCCESS = "cluster_manager.cds.update_success"
	CDS_UPDATE_REJECT  = "cluster_manager.cds.update_rejected"
	LDS_UPDATE_SUCCESS = "listener_manager.lds.update_success"
	LDS_UPDATE_REJECT  = "listener_manager.lds.update_rejected"
)

func statsForIstio(w http.ResponseWriter, r *http.Request) {
	sb := bytes.NewBufferString("")
	sb.WriteString(fmt.Sprintf("%s: %d\n", CDS_UPDATE_SUCCESS, conv.Stats.CdsUpdateSuccess.Count()))
	sb.WriteString(fmt.Sprintf("%s: %d\n", CDS_UPDATE_REJECT, conv.Stats.CdsUpdateReject.Count()))
	sb.WriteString(fmt.Sprintf("%s: %d\n", LDS_UPDATE_SUCCESS, conv.Stats.LdsUpdateSuccess.Count()))
	sb.WriteString(fmt.Sprintf("%s: %d\n", LDS_UPDATE_REJECT, conv.Stats.LdsUpdateReject.Count()))
	_, err := sb.WriteTo(w)

	if err != nil {
		log.DefaultLogger.Warnf("%v", err)
	}
}

func serverInfoForIstio(w http.ResponseWriter, r *http.Request) {
	mosnState := store.GetMosnState()

	i := envoyControlPlaneAPI.ServerInfo{}

	switch mosnState {
	case store.Active_Reconfiguring:
		i.State = envoyControlPlaneAPI.ServerInfo_PRE_INITIALIZING
	case store.Init:
		i.State = envoyControlPlaneAPI.ServerInfo_INITIALIZING
	case store.Running:
		i.State = envoyControlPlaneAPI.ServerInfo_LIVE
	case store.Passive_Reconfiguring:
		i.State = envoyControlPlaneAPI.ServerInfo_DRAINING
	}

	m := jsonpb.Marshaler{}
	err := m.Marshal(w, &i)
	if err != nil {
		log.DefaultLogger.Warnf("marshal to string failed")
	}
}
