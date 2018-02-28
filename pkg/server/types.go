package server

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	)

type Server interface {
	AddListener(lc v2.ListenerConfig)

	Start()

	Restart()

	Close()
}

type NetworkFilterFactoryCb func(manager types.FilterManager)

type NetworkFilterConfigFactory interface {
	CreateFilterFactory(clusterManager types.ClusterManager) NetworkFilterFactoryCb
}
