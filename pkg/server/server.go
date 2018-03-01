package server

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
)

type server struct {
	stopChan       chan bool
	listenersConfs []v2.ListenerConfig
	handler        types.ConnectionHandler
}

func NewServer(filterFactory NetworkFilterConfigFactory, cmFilter ClusterManagerFilter) Server {
	return &server{
		handler: NewHandler(filterFactory, cmFilter),
	}
}

func (src *server) AddListener(lc v2.ListenerConfig) {
	src.listenersConfs = append(src.listenersConfs, lc)
}

func (srv *server) AddOrUpdateListener(lc v2.ListenerConfig) {
	// TODO: support add listener or update existing listener
}

func (srv *server) Start() {
	// TODO: handle main thread panic

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
