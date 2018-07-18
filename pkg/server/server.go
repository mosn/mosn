/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package server

import (
	"errors"
	"os"
	"runtime"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/orcaman/concurrent-map"
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
	stopChan      chan struct{}
	handler       types.ConnectionHandler
	ListenerInMap cmap.ConcurrentMap
}

func NewServer(config *Config, cmFilter types.ClusterManagerFilter, clMng types.ClusterManager) Server {

	procNum := runtime.NumCPU()

	if config != nil {
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

	OnProcessShutDown(log.CloseAll)

	server := &server{
		logger:        log.DefaultLogger,
		stopChan:      make(chan struct{}),
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
		al := srv.handler.AddListener(lc, networkFiltersFactory, streamFiltersFactories)

		if activeListener, ok := al.(*activeListener); ok {
			go activeListener.listener.Start(nil)
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
		case <-srv.stopChan:
			return
		}
	}
}

func (src *server) Restart() {
	// TODO
}

func (srv *server) Close() {
	// stop listener and connections
	srv.handler.StopListeners(nil, true)

	close(srv.stopChan)
}

func Stop() {
	for _, server := range servers {
		server.Close()
	}
}

func StopAccept() {
	for _, server := range servers {
		server.handler.StopListeners(nil, false)
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

func InitDefaultLogger(config *Config) {

	var logPath string
	var logLevel log.LogLevel

	if config != nil {
		logPath = config.LogPath
		logLevel = config.LogLevel
	}

	//use default log path
	if logPath == "" {
		logPath = MosnLogDefaultPath
	}

	err := log.InitDefaultLogger(logPath, logLevel)
	if err != nil {
		log.StartLogger.Fatalln("initialize default logger failed : ", err)
	}
}
