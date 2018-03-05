package proxy

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/proxy/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
)

type RpcProxyFilterConfigFactory struct {
	Proxy *v2.RpcProxy
}
func (rpcf *RpcProxyFilterConfigFactory) CreateFilterFactory(clusterManager types.ClusterManager) server.NetworkFilterFactoryCb {
	return func(manager types.FilterManager) {
		manager.AddReadFilter(sofarpc.NewRPCProxy(rpcf.Proxy, clusterManager))
	}
}
