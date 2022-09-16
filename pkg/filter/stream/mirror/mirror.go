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

package mirror

import (
	"context"
	"net"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"
	"mosn.io/pkg/variable"
)

type mirror struct {
	amplification  int
	broadcast      bool
	receiveHandler api.StreamReceiverFilterHandler
	dp             api.ProtocolName
	up             api.ProtocolName
	ctx            context.Context
	headers        api.HeaderMap
	data           buffer.IoBuffer
	trailers       api.HeaderMap
	clusterName    string
	cluster        types.ClusterInfo
	sender         types.StreamSender
	host           types.Host
}

func (m *mirror) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	m.receiveHandler = handler
}

func (m *mirror) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {

	if m.receiveHandler.Route() == nil || m.receiveHandler.Route().RouteRule() == nil {
		return api.StreamFilterContinue
	}

	mirrorPolicy := m.receiveHandler.Route().RouteRule().Policy().MirrorPolicy()
	if !mirrorPolicy.IsMirror() {
		return api.StreamFilterContinue
	}

	clusterName := mirrorPolicy.ClusterName()

	utils.GoWithRecover(func() {
		clusterAdapter := cluster.GetClusterMngAdapterInstance()

		m.ctx = buffer.CleanBufferPoolContext(ctx)
		if headers != nil {
			// ! xprotocol should reimplement Clone function, not use default, trans protocol.CommonHeader
			h := headers.Clone()
			// nolint
			if _, ok := h.(protocol.CommonHeader); ok {
				log.DefaultLogger.Errorf("not support mirror, protocal {%v} must implement Clone function", m.getDownStreamProtocol())
				return
			}
			m.headers = h
		}
		if buf != nil {
			m.data = buf.Clone()
		}
		if trailers != nil {
			m.trailers = trailers.Clone()
		}

		m.dp, m.up = m.getProtocol()

		snap := clusterAdapter.GetClusterSnapshot(ctx, clusterName)
		if snap == nil {
			log.DefaultLogger.Errorf("mirror cluster {%s} not found", clusterName)
			return
		}
		m.cluster = snap.ClusterInfo()
		m.clusterName = clusterName

		amplification := m.amplification
		if m.broadcast {
			amplification = 0
			snap.HostSet().Range(func(host types.Host) bool {
				if host.Health() {
					amplification++
				}
				return true
			})
		}

		for i := 0; i < amplification; i++ {
			connPool, host := clusterAdapter.ConnPoolForCluster(m, snap, m.up)
			if connPool == nil {
				if log.DefaultLogger.GetLogLevel() >= log.INFO {
					log.DefaultLogger.Infof("mirror get connPool failed, cluster:%s", m.clusterName)
				}
				break
			}
			var (
				streamSender types.StreamSender
				failReason   types.PoolFailureReason
			)

			if m.up == protocol.HTTP1 {
				// ! http1 use fake receiver reduce connect
				_, streamSender, failReason = connPool.NewStream(m.ctx, &receiver{})
			} else {
				_, streamSender, failReason = connPool.NewStream(m.ctx, nil)
			}

			if failReason != "" {
				m.OnFailure(failReason, host)
				continue
			}

			m.OnReady(streamSender, host)
		}
	}, nil)
	if m.broadcast {
		m.receiveHandler.SendHijackReply(api.SuccessCode, m.headers)
		return api.StreamFilterStop
	}
	return api.StreamFilterContinue
}

func (m *mirror) OnDestroy() {}

func (m *mirror) getProtocol() (dp, up types.ProtocolName) {
	dp = m.getDownStreamProtocol()
	up = m.getUpstreamProtocol()
	return
}

func (m *mirror) getDownStreamProtocol() (prot types.ProtocolName) {
	if dpv, err := variable.Get(m.ctx, types.VariableDownStreamProtocol); err == nil {
		if dp, ok := dpv.(types.ProtocolName); ok {
			return dp
		}
	}
	return m.receiveHandler.RequestInfo().Protocol()
}

func (m *mirror) getUpstreamProtocol() (currentProtocol types.ProtocolName) {
	configProtocol := protocol.Auto

	if m.receiveHandler.Route() != nil && m.receiveHandler.Route().RouteRule() != nil && m.receiveHandler.Route().RouteRule().UpstreamProtocol() != "" {
		configProtocol = types.ProtocolName(m.receiveHandler.Route().RouteRule().UpstreamProtocol())
	}

	if protov, err := variable.Get(m.ctx, types.VariableUpstreamProtocol); err == nil {
		if proto, ok := protov.(types.ProtocolName); ok {
			configProtocol = proto
		}
	}

	if configProtocol == protocol.Auto {
		currentProtocol = m.getDownStreamProtocol()
	}
	return currentProtocol
}

func (m *mirror) MetadataMatchCriteria() api.MetadataMatchCriteria {
	return nil
}

func (m *mirror) DownstreamConnection() net.Conn {
	return m.receiveHandler.Connection().RawConn()
}

func (m *mirror) DownstreamHeaders() types.HeaderMap {
	return m.headers
}

func (m *mirror) DownstreamContext() context.Context {
	return m.ctx
}

func (m *mirror) DownstreamCluster() types.ClusterInfo {
	return m.cluster
}

func (m *mirror) DownstreamRoute() api.Route {
	return m.receiveHandler.Route()
}

func (m *mirror) OnFailure(reason types.PoolFailureReason, host types.Host) {}

func (m *mirror) OnReady(sender types.StreamSender, host types.Host) {
	m.sender = sender
	m.host = host

	m.sendDataOnce()
}

func (m *mirror) sendDataOnce() {
	endStream := m.data == nil && m.trailers == nil

	m.sender.AppendHeaders(m.ctx, m.headers, endStream)

	if endStream {
		return
	}

	endStream = m.trailers == nil
	m.sender.AppendData(m.ctx, m.data, endStream)

	if endStream {
		return
	}

	m.sender.AppendTrailers(m.ctx, m.trailers)
}
