package network

import (
	"context"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/network/faultinject"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/network/tcpproxy"
)

type FaultInjectFilterConfigFactory struct {
	FaultInject *v2.FaultInject
	Proxy       *v2.TcpProxy
}

func (fifcf *FaultInjectFilterConfigFactory) CreateFilterFactory(clusterManager types.ClusterManager,
	context context.Context) server.NetworkFilterFactoryCb {
	return func(manager types.FilterManager) {
		manager.AddReadFilter(faultinject.NewFaultInjecter(fifcf.FaultInject))
		manager.AddReadFilter(tcpproxy.NewProxy(fifcf.Proxy, clusterManager))
	}
}
