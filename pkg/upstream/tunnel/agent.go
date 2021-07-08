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

package tunnel

import (
	"encoding/json"
	"net"
	"sync"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/server"
	"mosn.io/mosn/pkg/upstream/tunnel/ext"
	"mosn.io/pkg/utils"
)

type agentBootstrapConfig struct {
	Enable bool `json:"enable"`
	// 建立连接数
	ConnectionNum int `json:"connection_num"`
	// 对应cluster的name
	Cluster string `json:"cluster"`
	// 处理listener name
	HostingListener string `json:"hosting_listener"`
	// Server侧的直连列表
	StaticServerList []string `json:"server_list"`

	DynamicServerListConfig struct {
		DynamicServerLister string `json:"dynamic_server_lister"`
	}

	// ConnectRetryTimes
	ConnectRetryTimes int `json:"connect_retry_times"`
	// ReconnectBaseDuration
	ReconnectBaseDurationMs int `json:"reconnect_base_duration_ms"`

	// ConnectTimeoutDurationMs
	ConnectTimeoutDurationMs int    `json:"connect_timeout_duration_ms"`
	CredentialPolicy         string `json:"credential_policy"`
}

func init() {
	v2.RegisterParseExtendConfig("tunnel_agent_bootstrap_config", func(config json.RawMessage) error {
		var conf agentBootstrapConfig
		err := json.Unmarshal(config, &conf)
		if err != nil {
			log.DefaultLogger.Errorf("[tunnel agent] failed to parse agent bootstrap config: %v", err.Error())
			return err
		}
		if conf.Enable {
			bootstrap(&conf)
		}
		return nil
	})
}

func bootstrap(conf *agentBootstrapConfig) {
	if conf.DynamicServerListConfig.DynamicServerLister != "" {
		utils.GoWithRecover(func() {
			lister := ext.GetServerLister(conf.DynamicServerListConfig.DynamicServerLister)
			ch := lister.List(conf.Cluster)
			select {
			case servers := <-ch:
				// Compute the diff between new and old server list
				intersection := make(map[string]bool)
				for i := range servers {
					if _, ok := connectionMap.Load(servers[i]); ok {
						intersection[servers[i]] = true
					}
				}
				increased := make([]string, 0)
				for _, addr := range servers {
					if _, ok := intersection[addr]; !ok {
						increased = append(increased, addr)
						utils.GoWithRecover(func() {
							connectServer(conf, addr)
						}, nil)
					}
				}
				decreased := make([]string, 0)
				connectionMap.Range(func(key, value interface{}) bool {
					addr := key.(string)
					_, ok := intersection[addr]
					if !ok {
						decreased = append(decreased, addr)
					}
					return true
				})
				for _, addr := range decreased {
					val, ok := connectionMap.LoadAndDelete(addr)
					if !ok {
						continue
					}
					for _, conn := range val.([]*AgentRawConnection) {
						err := conn.Stop()
						if err != nil {
							log.DefaultLogger.Errorf("[tunnel agent] failed to stop connection, err: %+v", err)
						}
					}
				}
				log.DefaultLogger.Infof("[tunnel agent] tunnel server list changed, update success, increased: %+v, decreased: %+v", increased, decreased)
			}
		}, nil)
	}

	for _, serverAddress := range conf.StaticServerList {
		host, port, err := net.SplitHostPort(serverAddress)
		if err != nil {
			return
		}
		addrs, err := net.LookupHost(host)
		if err != nil {
			log.DefaultLogger.Errorf("[tunnel agent] failed to lookup host by domain: %v", host)
			return
		}
		for _, addr := range addrs {
			utils.GoWithRecover(func() {
				connectServer(conf, net.JoinHostPort(addr, port))
			}, nil)
		}
	}
}

var connectionMap = &sync.Map{}

func connectServer(conf *agentBootstrapConfig, address string) {
	listener := server.GetServer().Handler().FindListenerByName(conf.HostingListener)
	if listener == nil {
		return
	}
	config := &ConnectionConfig{
		Address:                address,
		ClusterName:            conf.Cluster,
		Weight:                 10,
		ReconnectBaseDuration:  time.Duration(conf.ReconnectBaseDurationMs) * time.Millisecond,
		ConnectTimeoutDuration: time.Duration(conf.ConnectTimeoutDurationMs) * time.Millisecond,
		ConnectRetryTimes:      conf.ConnectRetryTimes,
		CredentialPolicy:       conf.CredentialPolicy,
	}
	connList := make([]*AgentRawConnection, 0, conf.ConnectionNum)
	for i := 0; i < conf.ConnectionNum; i++ {
		conn := NewConnection(*config, listener)
		err := conn.connectAndInit()
		if err == nil {
			connList = append(connList, conn)
		}
	}
	connectionMap.Store(address, connList)
}

type ConnectionConfig struct {
	Address           string `json:"address"`
	ClusterName       string `json:"cluster_name"`
	Weight            int64  `json:"weight"`
	ConnectRetryTimes int    `json:"connect_retry_times"`
	// 连接超时时间
	ConnectTimeoutDuration time.Duration `json:"connect_timeout_duration"`
	Network                string        `json:"network"`
	ReconnectBaseDuration  time.Duration `json:"reconnect_base_duration"`
	CredentialPolicy       string        `json:"credential_policy"`
}
