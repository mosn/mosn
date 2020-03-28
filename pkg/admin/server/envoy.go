package server

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/golang/protobuf/jsonpb"
	"mosn.io/mosn/pkg/admin/store"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/xds/conv"
	envoyControlPlaneAPI "github.com/envoyproxy/go-control-plane/envoy/admin/v2alpha"

)

func statsForEnvoy(w http.ResponseWriter, r *http.Request) {
	sb := bytes.NewBufferString("")
	sb.WriteString(fmt.Sprintf("cluster_manager.cds.update_success: %d\n", conv.Stats.CdsUpdateSuccess.Count()))
	sb.WriteString(fmt.Sprintf("cluster_manager.cds.update_rejected: %d\n", conv.Stats.CdsUpdateReject.Count()))
	sb.WriteString(fmt.Sprintf("listener_manager.lds.update_success: %d\n", conv.Stats.LdsUpdateSuccess.Count()))
	sb.WriteString(fmt.Sprintf("listener_manager.lds.update_rejected: %d\n", conv.Stats.LdsUpdateReject.Count()))
	_, err := sb.WriteTo(w)

	if err != nil {
		log.DefaultLogger.Warnf("%v", err)
	}
}

func serverInfoForEnvoy(w http.ResponseWriter, r *http.Request) {
	mosnState := store.GetMosnState()

	i := envoyControlPlaneAPI.ServerInfo{
	}

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
	err:=  m.Marshal(w, &i)
	if err != nil {
		log.DefaultLogger.Warnf("marshal to string failed")
	}
}
