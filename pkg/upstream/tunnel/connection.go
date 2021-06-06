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
	"fmt"
	"net"
	"time"

	"go.uber.org/atomic"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/tunnel/ext"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"
)

var (
	defaultReconnectBaseDuration = time.Second * 3
	defaultConnectMaxRetryTimes  = -1
)

type AgentRawConnection struct {
	ConnectionConfig
	rawc     net.Conn
	listener types.Listener
	close    *atomic.Bool
	initInfo *ConnectionInitInfo
}

func NewConnection(config ConnectionConfig, listener types.Listener) *AgentRawConnection {
	if config.Network == "" {
		config.Network = "tcp"
	}
	if config.ReconnectBaseDuration == 0 {
		config.ReconnectBaseDuration = defaultReconnectBaseDuration
	}
	if config.ConnectRetryTimes == 0 {
		config.ConnectRetryTimes = defaultConnectMaxRetryTimes
	}

	initInfo := &ConnectionInitInfo{
		ClusterName:      config.ClusterName,
		Weight:           config.Weight,
		CredentialPolicy: config.CredentialPolicy,
	}

	if config.CredentialPolicy != "" {
		credentialGetter := ext.GetConnectionCredentialGetter(config.CredentialPolicy)
		if credentialGetter == nil {
			panic("credential getter not found")
		}
		initInfo.Credential = credentialGetter(config.ClusterName)
	}

	return &AgentRawConnection{
		ConnectionConfig: config,
		listener:         listener,
		initInfo:         initInfo,
		close:            atomic.NewBool(false),
	}
}

func (a *AgentRawConnection) Stop() error {
	a.close.Store(true)
	if a.rawc == nil {
		return nil
	}
	return a.rawc.Close()
}
func (a *AgentRawConnection) connectAndInit() error {
	var rawc net.Conn
	var err error
	backoffConnectDuration := a.ReconnectBaseDuration

	for i := 0; i < a.ConnectRetryTimes || a.ConnectRetryTimes == -1; i++ {
		if a.close.Load() {
			return fmt.Errorf("connection closed, don't attempt to connect, address: %+v\", a.ConnectionConfig.Address")
		}
		rawc, err = net.Dial(a.Network, a.Address)
		rawc.SetReadDeadline(time.Now().Add(time.Second * 10))
		if err == nil {
			b, err := Encode(a.initInfo)
			if err != nil {
				continue
			}
			// write connection init request
			_, err = rawc.Write(b.Bytes())
			readBuffer := buffer.GetIoBuffer(1024)
			_, err = readBuffer.ReadOnce(rawc)
			if err != nil {
				log.DefaultLogger.Errorf("[agent] read response failed, err: %+v", a.Address, err)
				rawc.Close()
				continue
			}
			ret, err := DecodeFromBuffer(readBuffer)
			if err != nil || ret == nil {
				log.DefaultLogger.Warnf("[agent] decode from buffer failed, err: %+v", err)
				continue
			}
			resp := ret.(ConnectionInitResponse)
			if resp.Status != ConnectSuccess {
				// reconnect and write again
				log.DefaultLogger.Errorf("[agent] failed to write connection info to remote server, address: %v, status: %v", a.Address, resp.Status)
				// close connection and reconnect again
				rawc.Close()
				continue
			}
			a.rawc = rawc
			break
		}
		log.DefaultLogger.Errorf("[agent] failed to connect remote server, try again after %v seconds, address: %v, err: %+v", backoffConnectDuration, a.Address, err)
		time.Sleep(backoffConnectDuration)
		backoffConnectDuration *= 2
	}
	if err != nil {
		return err
	}

	// hosting new connection
	utils.GoWithRecover(func() {
		a.listener.GetListenerCallbacks().OnAccept(rawc, a.listener.UseOriginalDst(), nil, nil, nil, []api.ConnectionEventListener{a})
	}, nil)

	return nil
}

func (a *AgentRawConnection) OnEvent(event api.ConnectionEvent) {
	switch {
	case event.IsClose():
		goto RECONNECT
	case event.ConnectFailure():
		goto RECONNECT
	default:
		return
	}

RECONNECT:
	log.DefaultLogger.Infof("[agent] receive reconnect event, and try to reconnect remote server %v", a.Address)
	err := a.connectAndInit()
	if err != nil {
		log.DefaultLogger.Errorf("[agent] failed to reconnect remote server: %v", a.Address)
		return
	}
	log.DefaultLogger.Debugf("[agent] reconnect remote server: %v success", a.Address)
}
