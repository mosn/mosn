package proxy

import (
	"context"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/network/tcpproxy"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type TcpProxyFilterConfigFactory struct {
	Proxy *v2.TcpProxy
}

func (tpcf *TcpProxyFilterConfigFactory) CreateFilterFactory(clusterManager types.ClusterManager, context context.Context) types.NetworkFilterFactoryCb {
	return func(manager types.FilterManager) {
		manager.AddReadFilter(tcpproxy.NewProxy(tpcf.Proxy, clusterManager, context))
	}
}
