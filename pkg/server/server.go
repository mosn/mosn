package server

import (
	"errors"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"os"
	"runtime"
	_ "sync"
	"time"
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
	logger   log.Logger
	stopChan chan bool
	handler  types.ConnectionHandler
}

func NewServer(config *Config, cmFilter types.ClusterManagerFilter, clMng types.ClusterManager) Server {
	var logPath string
	var logLevel log.LogLevel
	procNum  := runtime.NumCPU()

	if config != nil {
		logPath = config.LogPath
		logLevel = config.LogLevel

		//graceful timeout setting
		if config.GracefulTimeout != 0 {
			gracefulTimeout = config.GracefulTimeout
		}

		//processor num setting
		if config.Processor > 0 {
			procNum = config.Processor
		}
	}


	runtime.GOMAXPROCS(procNum)

	initDefaultLogger(logPath, logLevel)

	OnProcessShutDown(log.CloseAll)

	server := &server{
		logger:   log.DefaultLogger,
		stopChan: make(chan bool),
		handler:  NewHandler(cmFilter, clMng, log.DefaultLogger),
	}

	servers = append(servers, server)

	return server
}

func (srv *server) AddListener(lc *v2.ListenerConfig, networkFiltersFactory types.NetworkFilterChainFactory, streamFiltersFactories []types.StreamFilterChainFactory) {
	srv.handler.AddListener(lc, networkFiltersFactory, streamFiltersFactories)
}

func (srv *server) AddOrUpdateListener(lc v2.ListenerConfig) {
	// TODO: support add listener or update existing listener
}

func (srv *server) Start() {
	// TODO: handle main thread panic @wugou

	srv.handler.StartListeners(nil)

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

func StopAccept() {
	for _, server := range servers {
		server.handler.StopListeners(nil)
	}
}

func ListListenerFD() []uintptr {
	var fds []uintptr
	for _, server := range servers {
		fds = append(fds, server.handler.ListListenersFD(nil)...)
	}
	return fds
}

func WaitConnectionsDone(duration time.Duration) error {
	timeout := time.NewTimer(duration)
	wait := make(chan struct{})
	go func() {
		//todo close idle connections and wait active connections complete
		time.Sleep(duration * 2)
		wait <- struct{}{}
	}()

	select {
	case <-timeout.C:
		return errors.New("wait timeout")
	case <-wait:
		return nil
	}
}

func initDefaultLogger(logPath string, logLevel log.LogLevel) {

	//use default log path
	if logPath == "" {
		logPath = MosnLogDefaultPath
	}

	err := log.InitDefaultLogger(logPath, logLevel)
	if err != nil {
		log.StartLogger.Fatalln("initialize default logger failed : ", err)
	}
}
