package server

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

type Config struct {
	LogPath  string
	LogLevel log.LogLevel
	// only used in http2 case
	DisableConnIo bool
}

type Server interface {
	AddListener(lc *v2.ListenerConfig, networkFiltersFactory types.NetworkFilterChainFactory, streamFiltersFactories []types.StreamFilterChainFactory)

	Start()

	Restart()

	Close()
}
