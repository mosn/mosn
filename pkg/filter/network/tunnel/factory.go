package tunnel

import (
	"context"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/upstream/cluster"
)

func init() {
	api.RegisterNetwork(v2.TUNNEL, CreateTunnelNetworkFilterFactory)
}

type tunnelNetworkFilterFactory struct {
	FaultInject *v2.FaultInject
}

func (f *tunnelNetworkFilterFactory) CreateFilterChain(context context.Context, callbacks api.NetWorkFilterChainFactoryCallbacks) {
	rf := &tunnelFilter{
		clusterManager: cluster.GetClusterMngAdapterInstance().ClusterManager,
	}
	callbacks.AddReadFilter(rf)
}

func CreateTunnelNetworkFilterFactory(config map[string]interface{}) (api.NetworkFilterChainFactory, error) {
	return &tunnelNetworkFilterFactory{}, nil
}
