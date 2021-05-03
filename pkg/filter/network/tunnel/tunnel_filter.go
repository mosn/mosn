package tunnel

import (
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/tunnel"
)

var _ api.ReadFilter = (*tunnelFilter)(nil)

type tunnelFilter struct {
	clusterManager  types.ClusterManager
	readCallbacks   api.ReadFilterCallbacks
	connInitialized bool
}

func (t *tunnelFilter) OnData(buffer api.IoBuffer) api.FilterStatus {
	if t.connInitialized {
		return api.Continue
	}
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[tunnel server] [ondata] read data , len = %v", buffer.Len())
	}
	data := tunnel.Read(buffer)
	if data != nil {
		// now it can only be ConnectionInitInfo
		info, ok := data.(*tunnel.ConnectionInitInfo)
		if ok {
			//
			conn := t.readCallbacks.Connection()
			conn.AddConnectionEventListener(NewHostRemover(conn.RemoteAddr().String(), info.ClusterName))
			if t.clusterManager.ClusterExist(info.ClusterName) {
				// set the flag that has been initialized, subsequent data processing skips this filter
				t.connInitialized = true
				t.clusterManager.AppendHostWithConnection(info.ClusterName, v2.Host{
					HostConfig: v2.HostConfig{
						Address:        conn.RemoteAddr().String(),
						Hostname:       info.HostName,
						Weight:         uint32(info.Weight),
						TLSDisable:     false,
					},
				}, network.CreateTunnelAgentConnection(conn))
			}
		}
	}
	return api.Continue
}

func (t *tunnelFilter) OnNewConnection() api.FilterStatus {
	return api.Continue
}

func (t *tunnelFilter) InitializeReadFilterCallbacks(cb api.ReadFilterCallbacks) {
	t.readCallbacks = cb
}
