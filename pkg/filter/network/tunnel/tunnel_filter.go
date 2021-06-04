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
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/tunnel"
	"mosn.io/mosn/pkg/upstream/tunnel/ext"
)

var _ api.ReadFilter = (*tunnelFilter)(nil)

// tunnelFilter handles the reverse connection sent by the edge node，
// mainly used in scenarios where cloud nodes and edge nodes may be located in separate and isolated network region.
type tunnelFilter struct {
	clusterManager types.ClusterManager
	readCallbacks  api.ReadFilterCallbacks
	// connInitialized is used as a flag to be processed only once
	connInitialized bool
}

func (t *tunnelFilter) OnData(buffer api.IoBuffer) api.FilterStatus {
	if t.connInitialized {
		// hand it over to the next filter directly
		return api.Continue
	}
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[tunnel server] [ondata] read data , len: %v", buffer.Len())
	}
	data, err := tunnel.DecodeFromBuffer(buffer)
	conn := t.readCallbacks.Connection()
	if err != nil || data == nil {
		log.DefaultLogger.Errorf("[tunnel server] [ondata] failed to read from buffer, close connection err: %+v", err)
		return writeConnectResponse(tunnel.ConnectUnknownFailed, conn)
	}
	// now it can only be ConnectionInitInfo
	info, ok := data.(tunnel.ConnectionInitInfo)
	if !ok {
		log.DefaultLogger.Errorf("[tunnel server] [ondata] decode failed, data error")
		return writeConnectResponse(tunnel.ConnectUnknownFailed, conn)
	}
	// Auth the connection
	if info.CredentialPolicy != "" {
		connCredential := ext.GetConnectionCredential(info.CredentialPolicy)
		if connCredential == nil {
			return writeConnectResponse(tunnel.ConnectAuthFailed, conn)
		}
		res := connCredential.ValidateCredential(info.Credential, info.HostName, info.ClusterName)
		if !res {
			return writeConnectResponse(tunnel.ConnectAuthFailed, conn)
		}
	}
	conn.AddConnectionEventListener(NewHostRemover(conn.RemoteAddr().String(), info.ClusterName))
	if !t.clusterManager.ClusterExist(info.ClusterName) {
		return writeConnectResponse(tunnel.ConnectClusterNotExist, conn)
	}
	// set the flag that has been initialized, subsequent data processing skips this filter
	t.connInitialized = true
	_ = t.clusterManager.AppendHostWithConnection(info.ClusterName, v2.Host{
		HostConfig: v2.HostConfig{
			Address:    conn.RemoteAddr().String(),
			Hostname:   info.HostName,
			Weight:     uint32(info.Weight),
			TLSDisable: false,
		},
	}, network.CreateTunnelAgentConnection(conn))
	return writeConnectResponse(tunnel.ConnectSuccess, conn)
}

func (t *tunnelFilter) OnNewConnection() api.FilterStatus {
	return api.Continue
}

func (t *tunnelFilter) InitializeReadFilterCallbacks(cb api.ReadFilterCallbacks) {
	t.readCallbacks = cb
}

func writeConnectResponse(status tunnel.ConnectStatus, conn api.Connection) api.FilterStatus {
	buffer, err := tunnel.Encode(&tunnel.ConnectionInitResponse{Status: status})
	if err != nil {
		conn.Close(api.NoFlush, api.LocalClose)
	}
	err = conn.Write(buffer)
	if err != nil {
		conn.Close(api.NoFlush, api.OnWriteErrClose)
	}
	return api.Stop
}
