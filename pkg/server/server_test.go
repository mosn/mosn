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
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	_ "mosn.io/mosn/pkg/filter/network/proxy"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mock"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	_ "mosn.io/mosn/pkg/proxy"
	"mosn.io/mosn/pkg/router"
	_ "mosn.io/mosn/test/util"
	"mosn.io/pkg/buffer"
)

func TestGracefulShutdown(t *testing.T) {
	r := require.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create server config
	v2ServerConf := configmanager.ParseServerConfig(&v2.ServerConfig{
		ServerName: "MockServer",
	})
	conf := NewConfig(v2ServerConf)

	// Create mock server dependency
	cmFilter := mock.NewMockClusterManagerFilter(ctrl)
	cmFilter.EXPECT().OnCreated(gomock.Any(), gomock.Any()).AnyTimes().Return()

	clusterMgr := mock.NewMockClusterManager(ctrl)
	clusterMgr.EXPECT().ConnPoolForCluster(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

	routeCfg := &v2.RouterConfiguration{
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: "test_router",
		},
	}
	err := router.GetRoutersMangerInstance().AddOrUpdateRouters(routeCfg)
	r.Nil(err)

	// Create server
	server := NewServer(conf, cmFilter, clusterMgr)

	// Prepare listener config
	confJson := `
	{
		"name": "bolt-egress",
		"address": "127.0.0.1:8668",
		"bind_port": true,
		"filter_chains": [{
			"filters": [
				{
					"type": "proxy",
					"config": {
						"downstream_protocol": "X",
						"upstream_protocol": "X",
						"extend_config": {
							"sub_protocol": "bolt",
							"bolt": {
								"enable_bolt_goaway": true
							}
						},
						"router_config_name":"test_router"
					}
				}
			]
		}],
		"stream_filters": []
	}
	`
	listenerConf := &v2.Listener{}
	err = json.Unmarshal([]byte(confJson), listenerConf)
	r.Nilf(err, "Unmarshal listener config fail, error: %v\n", err)

	// Add listener into server
	server.AddListener(listenerConf)

	// Start server
	go server.Start()
	defer server.Close()

	// Wait server started
	time.Sleep(50 * time.Millisecond)

	// Create connection to server
	conn, err := net.Dial("tcp", "127.0.0.1:8668")
	r.Nil(err, "Connect server fail")
	defer conn.Close()

	// Wait new connection processed
	time.Sleep(50 * time.Millisecond)

	// Shutdown server
	go server.Shutdown()

	// Server should send goaway frame on the exists connection
	var goawayChecker = func() bool {
		return checkHasSendGoaway(conn)
	}
	r.Eventually(goawayChecker, 300*time.Millisecond, 10*time.Millisecond, "Expect got a goaway frame")

	// Server should not accpect new connection.
	var connectNewConnChecker = func() bool {
		return !checkRefuseNewConnection()
	}
	r.Never(connectNewConnChecker, 300*time.Millisecond, 10*time.Millisecond, "Expect refused new connection")
}

func checkHasSendGoaway(conn net.Conn) bool {
	// Expect got goaway frame from server
	xCodec := bolt.XCodec{}
	bCodec := xCodec.NewXProtocol(context.Background())

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	buf := make([]byte, 10240)
	size, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		log.DefaultLogger.Errorf("Read data fail, error: %v", err)
		return false
	}

	value, err := bCodec.Decode(context.Background(), buffer.NewIoBufferBytes(buf[:size]))
	if err != nil {
		log.DefaultLogger.Errorf("Decode request fail, error: %v", err)
		return false
	}

	req, ok := value.(*bolt.Request)
	if !ok {
		log.DefaultLogger.Errorf("Request is not a request frame, type: %T", value)
		return false
	}

	if req.CmdCode != bolt.CmdCodeGoAway {
		log.DefaultLogger.Errorf("Expect goaway frame, cmdcode: %v", req.CmdCode)
		return false
	}

	return true
}

func checkRefuseNewConnection() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:8668", 10*time.Millisecond)
	if err == nil {
		conn.Close()
		log.DefaultLogger.Errorf("New connection connected")
		return false
	}

	opErr, ok := err.(*net.OpError)
	if !ok {
		log.DefaultLogger.Errorf("Error is not OpError, error type: %T", err)
		return false
	}

	syscallErr, ok := opErr.Err.(*os.SyscallError)
	if !ok {
		log.DefaultLogger.Errorf("Error is not SyscallError, error type: %T", err)
		return false
	}

	if syscallErr.Err != syscall.ECONNREFUSED {
		log.DefaultLogger.Errorf("Error is not connection refused, error type: %T, error: %v", err, err)
		return false
	}

	return true
}
