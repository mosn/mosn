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
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/upstream/cluster"
)

type clusterManagerFilterMocK struct {
	cccb types.ClusterConfigFactoryCb
	chcb types.ClusterHostFactoryCb
}

func (cmf *clusterManagerFilterMocK) OnCreated(cccb types.ClusterConfigFactoryCb, chcb types.ClusterHostFactoryCb) {
	cmf.cccb = cccb
	cmf.chcb = chcb
}

var srvAddresses = []string{"127.0.0.1:8080", "127.0.0.1:8081", "127.0.0.1:8082"}
var clnAddresses = []string{"127.0.0.1:9090", "127.0.0.1:9091", "127.0.0.1:9092"}

var once sync.Once
var stopServerChan = make(chan bool, 1)

func runMockServer(t *testing.T) {
	once.Do(func() {

		address, err := net.ResolveTCPAddr("tcp", srvAddresses[0])
		if err != nil {
			t.Errorf("resolve tcp address error, address = %s", address)
		}

		mockConfig := &Config{
			ServerName: "mock_server_1",
			LogPath:    "",
			LogLevel:   log.DEBUG,
		}

		log.InitDefaultLogger(mockConfig.LogPath, mockConfig.LogLevel)

		cmf := &clusterManagerFilterMocK{}
		cm := cluster.NewClusterManager(nil, nil, nil, true, false)
		mockServer := NewServer(mockConfig, cmf, cm)

		listenConfig := &v2.Listener{
			ListenerConfig: v2.ListenerConfig{
				Name:                                  "listener1",
				BindToPort:                            true,
				LogPath:                               "stdout",
				HandOffRestoredDestinationConnections: true,
			},
			Addr:                    address,
			PerConnBufferLimitBytes: 1 << 15,
			LogLevel:                3,
		}

		mockServer.AddListener(listenConfig, nil, nil)
		mockServer.Start()

		for {
			select {
			case <-stopServerChan:
				mockServer.Close()
			}
		}
	})
}

func runMockClientConnect(clientAddress string, serverAddress string) (net.Conn, error) {
	localTCPAddr, err1 := net.ResolveTCPAddr("tcp", clientAddress)
	if err1 != nil {
		return nil, fmt.Errorf("resolve tcp address error, clientaddress = %s", clientAddress)
	}

	remoteTCPAddr, err2 := net.ResolveTCPAddr("tcp", serverAddress)

	if err2 != nil {
		return nil, fmt.Errorf("resolve tcp address error, clientaddress = %s, server", serverAddress)
	}

	conn, err := net.DialTCP("tcp", localTCPAddr, remoteTCPAddr)
	return conn, err
}

func TestListenerAdapter_AddOrUpdateListener(t *testing.T) {
	go runMockServer(t)
	time.Sleep(1 * time.Second) // wait server start

	serverAddress := srvAddresses[1]
	localTCPAddr := clnAddresses[0]
	addedAddress, _ := net.ResolveTCPAddr("tcp", serverAddress) //added server

	addedListenerConfig := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:                                  "listener2",
			BindToPort:                            true,
			LogPath:                               "stdout",
			HandOffRestoredDestinationConnections: true,
		},
		Addr:                    addedAddress,
		PerConnBufferLimitBytes: 1 << 15,
		LogLevel:                3,
	}

	updateListenerConfig := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:                                  "listener2",
			BindToPort:                            false,
			LogPath:                               "stdout",
			HandOffRestoredDestinationConnections: true,
		},
		Addr:                    addedAddress,
		PerConnBufferLimitBytes: 1 << 15,
		LogLevel:                3,
	}

	type fields struct {
		connHandlerMap     map[string]types.ConnectionHandler
		defaultConnHandler types.ConnectionHandler
	}
	type args struct {
		serverName             string
		lc                     *v2.Listener
		networkFiltersFactory  []types.NetworkFilterChainFactory
		streamFiltersFactories []types.StreamFilterChainFactory
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "testListenerAdd",
			fields: fields{
				connHandlerMap:     GetListenerAdapterInstance().connHandlerMap,
				defaultConnHandler: GetListenerAdapterInstance().defaultConnHandler,
			},
			args: args{
				serverName:             "",
				lc:                     addedListenerConfig,
				networkFiltersFactory:  nil,
				streamFiltersFactories: nil,
			},
			wantErr: false,
		},
		{
			name: "testListenerUpdate",
			fields: fields{
				connHandlerMap:     GetListenerAdapterInstance().connHandlerMap,
				defaultConnHandler: GetListenerAdapterInstance().defaultConnHandler,
			},
			args: args{
				serverName:             "",
				lc:                     updateListenerConfig,
				networkFiltersFactory:  nil,
				streamFiltersFactories: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			adapter := &ListenerAdapter{
				connHandlerMap:     tt.fields.connHandlerMap,
				defaultConnHandler: tt.fields.defaultConnHandler,
			}

			if tt.name == "testListenerAdd" {
				// expect connect failure
				if adapter.defaultConnHandler.FindListenerByName("listener2") != nil {
					t.Errorf("listener = %s already in", srvAddresses[0])
				}

				if conn, err := runMockClientConnect(localTCPAddr, srvAddresses[1]); err == nil {
					t.Errorf("listener = %s already running, need check ", srvAddresses[1])
					conn.Close()
				}

				// do listener start
				if err := adapter.AddOrUpdateListener(tt.args.serverName, tt.args.lc, tt.args.networkFiltersFactory, tt.args.streamFiltersFactories); (err != nil) != tt.wantErr {
					t.Errorf("ListenerAdapter.AddOrUpdateListener() error = %v, wantErr %v", err, tt.wantErr)
				}

				time.Sleep(1 * time.Second) // wait listener start

				// expect connect success
				if adapter.defaultConnHandler.FindListenerByName("listener2") == nil {
					t.Errorf("listener = %s add listener error", srvAddresses[0])
				}

				if conn, err := runMockClientConnect(localTCPAddr, srvAddresses[1]); err != nil {
					t.Errorf("ListenerAdapter.AddOrUpdateListener() error = %v, wantErr %v", err, tt.wantErr)

				} else {
					conn.Close()
				}
			}

			if tt.name == "testListenerUpdate" {
				if err := adapter.AddOrUpdateListener(tt.args.serverName, tt.args.lc, tt.args.networkFiltersFactory, tt.args.streamFiltersFactories); (err != nil) != tt.wantErr {
					t.Errorf("ListenerAdapter.AddOrUpdateListener() error = %v, wantErr %v", err, tt.wantErr)
				}

				if listener := adapter.defaultConnHandler.FindListenerByName("listener2"); listener == nil {
					t.Errorf("testListenerUpdate error, listener don't exist")
				} else if listener.Config() != updateListenerConfig {
					t.Errorf("testListenerUpdate error, config remain the same")
				}
			}
		})
	}
}

func TestListenerAdapter_DeleteListener(t *testing.T) {
	go runMockServer(t)
	time.Sleep(1 * time.Second) // wait server start

	adapter := &ListenerAdapter{
		connHandlerMap:     GetListenerAdapterInstance().connHandlerMap,
		defaultConnHandler: GetListenerAdapterInstance().defaultConnHandler,
	}

	// expect connect success
	if adapter.defaultConnHandler.FindListenerByName("listener1") == nil {
		t.Errorf("listener = %s doesn't start ", srvAddresses[0])
	}

	if conn, err := runMockClientConnect(clnAddresses[1], srvAddresses[0]); err != nil {
		t.Errorf("ListenerAdapter.DeleteListener() error = %s ", err.Error())
	} else {
		conn.Close()
	}

	// delete the listener
	if err := adapter.DeleteListener("", "listener1"); err != nil {
		t.Errorf("ListenerAdapter.DeleteListener() error = %v", err.Error())
	}

	if adapter.defaultConnHandler.FindListenerByName("listener1") != nil {
		t.Errorf("listener = %s doesn't stop ", srvAddresses[0])
	}

	time.Sleep(3 * time.Second)

	// expect connect failure
	if conn, err := runMockClientConnect(clnAddresses[2], srvAddresses[0]); err == nil {
		t.Errorf("listener = %s doesn't stop ", srvAddresses[0])
		conn.Close()
	}
}
