package server

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"os"
	"time"
)

const (
	MosnBasePath = string(os.PathSeparator) + "home" + string(os.PathSeparator) +
		"admin" + string(os.PathSeparator) + "mosn"

	MosnLogBasePath = MosnBasePath + string(os.PathSeparator) + "logs"

	MosnLogDefaultPath = MosnLogBasePath + string(os.PathSeparator) + "mosn.log"

	MosnPidFileName = "mosn.pid"
)

type Config struct {
	LogPath         string
	LogLevel        log.LogLevel
	GracefulTimeout time.Duration
}

type Server interface {
	AddListener(lc *v2.ListenerConfig, networkFiltersFactory types.NetworkFilterChainFactory, streamFiltersFactories []types.StreamFilterChainFactory)

	Start()

	Restart()

	Close()
}
