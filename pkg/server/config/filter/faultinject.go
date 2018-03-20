package filter

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/faultinject"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream/tcp"
)

type FaultInjectFilterConfigFactory struct {
	FaultInject *v2.FaultInject
	Proxy       *v2.TcpProxy
}

func (fifcf *FaultInjectFilterConfigFactory) CreateFilterFactory(clusterManager types.ClusterManager) server.NetworkFilterFactoryCb {
	return func(manager types.FilterManager) {
		manager.AddReadFilter(faultinject.NewFaultInjecter(fifcf.FaultInject))
		manager.AddReadFilter(tcp.NewProxy(fifcf.Proxy, clusterManager))
	}
}
