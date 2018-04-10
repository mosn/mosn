package server

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"os"
)

func init() {
	onProcessExit = append(onProcessExit, func() {
		if pidFile != "" {
			os.Remove(pidFile)
		}
	})
}

var servers []*server

type server struct {
	logger         log.Logger
	stopChan       chan bool
	listenersConfs []*v2.ListenerConfig
	handler        types.ConnectionHandler
}

func NewServer(config *Config, networkFiltersFactory NetworkFilterChainFactory,
	streamFiltersFactories []types.StreamFilterChainFactory, cmFilter ClusterManagerFilter) Server {
	var logPath string
	var logLevel log.LogLevel
	var disableConnIo bool

	if config != nil {
		logPath = config.LogPath
		logLevel = config.LogLevel
		disableConnIo = config.DisableConnIo
	}

	log.InitDefaultLogger(logPath, logLevel)
	OnProcessShutDown(log.CloseAll)

	server := &server{
		logger:   log.DefaultLogger,
		stopChan: make(chan bool),
		handler:  NewHandler(networkFiltersFactory, streamFiltersFactories, cmFilter, log.DefaultLogger, disableConnIo),
	}

	servers = append(servers, server)

	return server
}

func (src *server) AddListener(lc *v2.ListenerConfig) {
	src.listenersConfs = append(src.listenersConfs, lc)
}

func (srv *server) AddOrUpdateListener(lc v2.ListenerConfig) {
	// TODO: support add listener or update existing listener
}

func (srv *server) Start() {
	// TODO: handle main thread panic @wugou

	for _, lc := range srv.listenersConfs {
		l := network.NewListener(lc)
		srv.handler.StartListener(l)
	}

	for {
		select {
		case stop := <-srv.stopChan:
			if stop {
				break
			}
		}
	}
}

func (src *server) Restart() {
	// TODO
}

func (srv *server) Close() {
	// stop listener and connections
	srv.handler.StopListeners(nil)

	srv.stopChan <- true
}

func Stop() {
	for _, server := range servers {
		server.Close()
	}
}
