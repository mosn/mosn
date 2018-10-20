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
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	onProcessExit = append(onProcessExit, func() {
		if pidFile != "" {
			os.Remove(pidFile)
		}
	})
}

// currently, only one server supported
func GetServer() Server {
	if len(servers) == 0 {
		log.DefaultLogger.Errorf("Server is nil and hasn't been initiated at this time")
		return nil
	}

	return servers[0]
}

var servers []*server

type server struct {
	serverName string
	logger     log.Logger
	stopChan   chan struct{}
	handler    types.ConnectionHandler
}

func NewServer(config *Config, cmFilter types.ClusterManagerFilter, clMng types.ClusterManager) Server {
	procNum := runtime.NumCPU()

	if config != nil {
		//graceful timeout setting
		if config.GracefulTimeout != 0 {
			GracefulTimeout = config.GracefulTimeout
		}

		//processor num setting
		if config.Processor > 0 {
			procNum = config.Processor
		}

		network.UseNetpollMode = config.UseNetpollMode
		if config.UseNetpollMode {
			log.StartLogger.Infof("Netpoll mode enabled.")
		}
	}

	runtime.GOMAXPROCS(procNum)

	OnProcessShutDown(log.CloseAll)

	server := &server{
		serverName: config.ServerName,
		logger:     log.DefaultLogger,
		stopChan:   make(chan struct{}),
		handler:    NewHandler(cmFilter, clMng, log.DefaultLogger),
	}

	initListenerAdapterInstance(server.serverName, server.handler)

	servers = append(servers, server)

	return server
}

func (srv *server) AddListener(lc *v2.Listener, networkFiltersFactories []types.NetworkFilterChainFactory,
	streamFiltersFactories []types.StreamFilterChainFactory) (types.ListenerEventListener, error) {

	return srv.handler.AddOrUpdateListener(lc, networkFiltersFactories, streamFiltersFactories)
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

func (srv *server) Restart() {
	// TODO
}

func (srv *server) Close() {
	// stop listener and connections
	srv.handler.StopListeners(nil, true)

	close(srv.stopChan)
}

func (srv *server) Handler() types.ConnectionHandler {
	return srv.handler
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

func StopConnection() {
	for _, server := range servers {
		server.handler.StopConnection()
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
	// one duration wait for connection to active close
	// two duration wait for connection to transfer
	// 10 seconds wait for read timeout
	timeout := time.NewTimer(duration*2 + time.Second*10)
	wait := make(chan struct{}, 1)
	time.Sleep(duration)
	go func() {
		//todo close idle connections and wait active connections complete
		StopConnection()
		log.DefaultLogger.Infof("StopConnection")
		time.Sleep(duration + time.Second*10)
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
	var logLevel log.Level

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
