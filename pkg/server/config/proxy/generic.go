package proxy

import (
	"context"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/proxy"
)

type GenericProxyFilterConfigFactory struct {
	Proxy *v2.Proxy
}

func (gfcf *GenericProxyFilterConfigFactory) CreateFilterFactory(clusterManager types.ClusterManager, context context.Context) types.NetworkFilterFactoryCb {
	return func(manager types.FilterManager) {
		manager.AddReadFilter(proxy.NewProxy(gfcf.Proxy, clusterManager, context))
	}
}