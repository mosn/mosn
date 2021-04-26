package tunnel

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/tunnel"
	"mosn.io/pkg/buffer"
)

var _ api.ReadFilter = (*tunnelFilter)(nil)

type tunnelFilter struct {
	clusterManager types.ClusterManager
	readCallbacks  api.ReadFilterCallbacks
}

func (t *tunnelFilter) OnData(buffer api.IoBuffer) api.FilterStatus {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[tunnel server] [ondata] read data , len = %v", p.network, buffer.Len())
	}
	data := tunnel.Read(buffer)
	if data != nil {
		info, ok := data.(tunnel.ConnectionInitInfo)
		if ok {
			conn := t.readCallbacks.Connection()
			t.clusterManager.AppendClusterHosts(info.ClusterName,v2.Ho)
		}

	}
	return api.Stop
}

func (t *tunnelFilter) OnNewConnection() api.FilterStatus {
}

func (t *tunnelFilter) InitializeReadFilterCallbacks(cb api.ReadFilterCallbacks) {
	t.readCallbacks = cb
}
