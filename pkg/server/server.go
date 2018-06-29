package server

import (
	"errors"
	"github.com/orcaman/concurrent-map"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
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

// currently, only one server supported
func GetServer() *server {
	if len(servers) == 0 {
		log.DefaultLogger.Errorf("Server is nil and hasn't been initiated at this time")
		return nil
	}

	return servers[0]
}

var servers []*server

type server struct {
	logger        log.Logger
	stopChan      chan bool
	handler       types.ConnectionHandler
	ListenerInMap cmap.ConcurrentMap
}

func NewServer(config *Config, cmFilter types.ClusterManagerFilter, clMng types.ClusterManager) Server {
	var logPath string
	var logLevel log.LogLevel
	procNum := runtime.NumCPU()

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
		logger:        log.DefaultLogger,
		stopChan:      make(chan bool),
		handler:       NewHandler(cmFilter, clMng, log.DefaultLogger),
		ListenerInMap: cmap.New(),
	}

	servers = append(servers, server)

	return server
}

func (srv *server) AddListener(lc *v2.ListenerConfig, networkFiltersFactory types.NetworkFilterChainFactory, streamFiltersFactories []types.StreamFilterChainFactory) {
	if srv.ListenerInMap.Has(lc.Name) {
		log.DefaultLogger.Warnf("Listen Already Started, Listen = %+v", lc)
	} else {
		srv.ListenerInMap.Set(lc.Name, lc)
		srv.handler.AddListener(lc, networkFiltersFactory, streamFiltersFactories)
	}
}

func (srv *server) AddListenerAndStart(lc *v2.ListenerConfig, networkFiltersFactory types.NetworkFilterChainFactory,
	streamFiltersFactories []types.StreamFilterChainFactory) error {

	if srv.ListenerInMap.Has(lc.Name) {
		log.DefaultLogger.Warnf("Listener Already Started, Listener Name = %+v", lc.Name)
	} else {
		srv.ListenerInMap.Set(lc.Name, lc)
		srv.handler.AddListener(lc, networkFiltersFactory, streamFiltersFactories)

		listenerStopChan := make(chan bool)

		//use default listener path
		if lc.LogPath == "" {
			lc.LogPath = MosnLogBasePath + string(os.PathSeparator) + lc.Name + ".log"
		}

		if ch, ok := srv.handler.(*connHandler); !ok {
			log.DefaultLogger.Errorf("Add Listener Error")
			return errors.New("Add Listener Error")
		} else {

			logger, err := log.NewLogger(lc.LogPath, log.LogLevel(lc.LogLevel))
			if err != nil {

				log.DefaultLogger.Errorf("initialize listener logger failed : %v", err)
				return err
			}

			//initialize access log
			var als []types.AccessLog

			for _, alConfig := range lc.AccessLogs {

				//use default listener access log path
				if alConfig.Path == "" {
					alConfig.Path = MosnLogBasePath + string(os.PathSeparator) + lc.Name + "_access.log"
				}

				if al, err := log.NewAccessLog(alConfig.Path, nil, alConfig.Format); err == nil {
					als = append(als, al)
				} else {
					log.DefaultLogger.Errorf("initialize listener access logger %s failed : %v", alConfig.Path, err)
					return err
				}
			}

			l := network.NewListener(lc, logger)

			al := newActiveListener(l, logger, als, networkFiltersFactory, streamFiltersFactories, ch, listenerStopChan, lc.DisableConnIo)
			l.SetListenerCallbacks(al)

			// start listener
			go al.listener.Start(nil)
		}
	}
	
	return nil
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
