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
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mosn"
	"mosn.io/mosn/pkg/upstream/tunnel/ext"
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

	CredentialPolicy string `json:"credential_policy"`
}

func init() {
	v2.RegisterParseExtendConfig("agent_bootstrap_config", func(config json.RawMessage) error {
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
		go func() {
			lister := ext.GetServerLister(conf.DynamicServerListConfig.DynamicServerLister)
			ch := lister.List(conf.Cluster)
			select {
			case servers := <-ch:
				// Compute the diff between new and old server list
				intersection := make(map[string]bool)
				for i := range servers {
					intersection[servers[i]] = true
				}
				for addr := range connectionMap {
					intersection[addr] = true
				}

				increased := make([]string, 0)
				for _, addr := range servers {
					if _, ok := intersection[addr]; !ok {
						increased = append(increased, addr)
						connectServer(conf, addr)
					}
				}

				decreased := make([]string, 0)
				for addr, conns := range connectionMap {
					_, ok := intersection[addr]
					if !ok {
						for _, conn := range conns {
							err := conn.Stop()
							if err != nil {
								log.DefaultLogger.Errorf("failed to")
							}
						}
						decreased = append(decreased, addr)
					}
				}
			}
		}()

	}

	for _, serverAddress := range conf.StaticServerList {
		host, port, err := net.SplitHostPort(serverAddress)
		if err != nil {
			return
		}
		addrs, err := net.LookupHost(host)
		if err != nil {
			log.DefaultLogger.Errorf("failed to lookup host by domain: %v", host)
			return
		}
		for _, addr := range addrs {
			go connectServer(conf, net.JoinHostPort(addr, port))
		}
	}
}

var connectionMap = map[string][]*AgentRawConnection{}

func connectServer(conf *agentBootstrapConfig, address string) {
	servers := mosn.MOSND.GetServer()
	listener := servers[0].Handler().FindListenerByName(conf.HostingListener)
	if listener == nil {
		return
	}
	config := &ConnectionConfig{
		Address:               address,
		ClusterName:           conf.Cluster,
		Weight:                10,
		ReconnectBaseDuration: time.Duration(conf.ReconnectBaseDurationMs) * time.Millisecond,
		ConnectRetryTimes:     conf.ConnectRetryTimes,
		CredentialPolicy:      conf.CredentialPolicy,
	}
	for i := 0; i < conf.ConnectionNum; i++ {
		conn := NewConnection(*config, listener)
		err := conn.connectAndInit()
		if err == nil {
			connectionMap[address] = append(connectionMap[address], conn)
		}
	}
}

type ConnectionConfig struct {
	Address               string        `json:"address"`
	ClusterName           string        `json:"cluster_name"`
	Weight                int64         `json:"weight"`
	ConnectRetryTimes     int           `json:"connect_retry_times"`
	Network               string        `json:"network"`
	ReconnectBaseDuration time.Duration `json:"reconnect_base_duration"`
	CredentialPolicy      string        `json:"credential_policy"`
}
