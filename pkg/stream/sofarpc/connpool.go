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
package sofarpc

import (
	"context"

	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	str "gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// types.ConnectionPool
type connPool struct {
	activeClient *activeClient
	host         types.Host
}

func NewConnPool(host types.Host) types.ConnectionPool {
	return &connPool{
		host: host,
	}
}

func (p *connPool) Protocol() types.Protocol {
	return protocol.SofaRpc
}

func (p *connPool) DrainConnections() {}

func (p *connPool) NewStream(context context.Context, streamId string,
	responseDecoder types.StreamReceiver, cb types.PoolEventListener) types.Cancellable {
	if p.activeClient == nil {
		p.activeClient = newActiveClient(context, p)
	}
	
	if p.activeClient == nil {
		cb.OnFailure(streamId, types.ConnectionFailure, nil)
	} else if !p.host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		cb.OnFailure(streamId, types.Overflow, nil)
	} else {
		// todo: update host stats
		p.activeClient.totalStream++
		p.host.ClusterInfo().ResourceManager().Requests().Increase()
		streamEncoder := p.activeClient.codecClient.NewStream(streamId, responseDecoder)
		cb.OnReady(streamId, streamEncoder, p.host)
	}

	return nil
}

func (p *connPool) Close() {
	if p.activeClient != nil {
		p.activeClient.codecClient.Close()
	}
}

func (p *connPool) onConnectionEvent(client *activeClient, event types.ConnectionEvent) {
	if event.IsClose() {
		// todo: update host stats
		p.activeClient = nil
	} else if event == types.ConnectTimeout {
		// todo: update host stats
		client.codecClient.Close()
	}
}

func (p *connPool) onStreamDestroy(client *activeClient) {
	// todo: update host stats
	p.host.ClusterInfo().ResourceManager().Requests().Decrease()
}

func (p *connPool) onStreamReset(client *activeClient, reason types.StreamResetReason) {
	// todo: update host stats
}

func (p *connPool) createCodecClient(context context.Context, connData types.CreateConnectionData) str.CodecClient {
	return str.NewCodecClient(context, protocol.SofaRpc, connData.Connection, connData.HostInfo)
}

// stream.CodecClientCallbacks
// types.ConnectionEventListener
// types.StreamConnectionEventListener
type activeClient struct {
	pool        *connPool
	codecClient str.CodecClient
	host        types.HostInfo
	totalStream uint64
}

func newActiveClient(context context.Context, pool *connPool) *activeClient {
	ac := &activeClient{
		pool: pool,
	}
	
	data := pool.host.CreateConnection(context)
	
	if err := data.Connection.Connect(true); err != nil {
		return nil
	}
	
	codecClient := pool.createCodecClient(context, data)
	codecClient.AddConnectionCallbacks(ac)
	codecClient.SetCodecClientCallbacks(ac)
	codecClient.SetCodecConnectionCallbacks(ac)

	ac.codecClient = codecClient
	ac.host = data.HostInfo
	
	return ac
}

func (ac *activeClient) OnEvent(event types.ConnectionEvent) {
	ac.pool.onConnectionEvent(ac, event)
}

func (ac *activeClient) OnStreamDestroy() {
	ac.pool.onStreamDestroy(ac)
}

func (ac *activeClient) OnStreamReset(reason types.StreamResetReason) {
	ac.pool.onStreamReset(ac, reason)
}

func (ac *activeClient) OnGoAway() {}
