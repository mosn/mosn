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

package http

import (
	"context"
	"sync"
	"time"

	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/network"
	"sofastack.io/sofa-mosn/pkg/protocol"
	str "sofastack.io/sofa-mosn/pkg/stream"
	"sofastack.io/sofa-mosn/pkg/types"
	"sofastack.io/sofa-mosn/pkg/utils"
)

//const defaultIdleTimeout = time.Second * 60 // not used yet

func init() {
	network.RegisterNewPoolFactory(protocol.HTTP1, NewConnPool)
	types.RegisterConnPoolFactory(protocol.HTTP1, true)
}

// types.ConnectionPool
type connPool struct {
	MaxConn int

	host types.Host

	statReport bool

	clientMux        sync.Mutex
	availableClients []*activeClient // available clients
	totalClientCount uint64          // total clients
}

func NewConnPool(host types.Host) types.ConnectionPool {
	pool := &connPool{
		host: host,
	}

	if pool.statReport {
		pool.report()
	}

	return pool
}

func (p *connPool) SupportTLS() bool {
	return p.host.SupportTLS()
}

func (p *connPool) Protocol() types.Protocol {
	return protocol.HTTP1
}

func (p *connPool) CheckAndInit(ctx context.Context) bool {
	return true
}

//由 PROXY 调用
func (p *connPool) NewStream(ctx context.Context, receiver types.StreamReceiveListener, listener types.PoolEventListener) {
	c, reason := p.getAvailableClient(ctx)

	if c == nil {
		listener.OnFailure(reason, p.host)
		return
	}

	if !p.host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		listener.OnFailure(types.Overflow, p.host)
		p.host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
	} else {
		p.host.HostStats().UpstreamRequestTotal.Inc(1)
		p.host.HostStats().UpstreamRequestActive.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestTotal.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestActive.Inc(1)
		p.host.ClusterInfo().ResourceManager().Requests().Increase()

		streamEncoder := c.client.NewStream(ctx, receiver)
		streamEncoder.GetStream().AddEventListener(c)
		listener.OnReady(streamEncoder, p.host)
	}

	return
}

func (p *connPool) getAvailableClient(ctx context.Context) (*activeClient, types.PoolFailureReason) {
	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	n := len(p.availableClients)
	// no available client
	if n == 0 {
		// max conns is 0 means no limit
		maxConns := p.host.ClusterInfo().ResourceManager().Connections().Max()
		if maxConns == 0 || p.totalClientCount < maxConns {
			ac, reason := newActiveClient(ctx, p)
			if ac != nil && reason == "" {
				p.totalClientCount++
			}
			return ac, reason
		} else {
			p.host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
			p.host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
			return nil, types.Overflow
		}
	} else {
		n--
		c := p.availableClients[n]
		p.availableClients[n] = nil
		p.availableClients = p.availableClients[:n]
		return c, ""
	}
}

func (p *connPool) Close() {
	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	for _, c := range p.availableClients {
		c.client.Close()
	}
}

func (p *connPool) Shutdown() {
	// TODO: http connpool do nothing for shutdown
}

func (p *connPool) onConnectionEvent(client *activeClient, event types.ConnectionEvent) {
	if event.IsClose() {

		if client.closeWithActiveReq {
			if event == types.LocalClose {
				p.host.HostStats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
				p.host.ClusterInfo().Stats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
			} else if event == types.RemoteClose {
				p.host.HostStats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)
				p.host.ClusterInfo().Stats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)
			}
		}

		// check if closed connection is available
		p.clientMux.Lock()
		defer p.clientMux.Unlock()

		p.totalClientCount--

		for i, c := range p.availableClients {
			if c == client {
				p.availableClients[i] = nil
				p.availableClients = append(p.availableClients[:i], p.availableClients[i+1:]...)
				break
			}
		}

		// set closed flag if not available
		client.closed = true
	} else if event == types.ConnectTimeout {
		p.host.HostStats().UpstreamRequestTimeout.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestTimeout.Inc(1)
		client.client.Close()
	} else if event == types.ConnectFailed {
		p.host.HostStats().UpstreamConnectionConFail.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamConnectionConFail.Inc(1)
	}
}

func (p *connPool) onStreamDestroy(client *activeClient) {
	p.host.HostStats().UpstreamRequestActive.Dec(1)
	p.host.ClusterInfo().Stats().UpstreamRequestActive.Dec(1)
	p.host.ClusterInfo().ResourceManager().Requests().Decrease()

	// return to pool
	p.clientMux.Lock()
	if !client.closed {
		p.availableClients = append(p.availableClients, client)
	}
	p.clientMux.Unlock()
}

func (p *connPool) onStreamReset(client *activeClient, reason types.StreamResetReason) {
	if reason == types.StreamConnectionTermination || reason == types.StreamConnectionFailed {
		p.host.HostStats().UpstreamRequestFailureEject.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestFailureEject.Inc(1)
		client.closeWithActiveReq = true
	} else if reason == types.StreamLocalReset {
		p.host.HostStats().UpstreamRequestLocalReset.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestLocalReset.Inc(1)
	} else if reason == types.StreamRemoteReset {
		p.host.HostStats().UpstreamRequestRemoteReset.Inc(1)
		p.host.ClusterInfo().Stats().UpstreamRequestRemoteReset.Inc(1)
	}
}

func (p *connPool) createStreamClient(context context.Context, connData types.CreateConnectionData) str.Client {
	return str.NewStreamClient(context, protocol.HTTP1, connData.Connection, connData.HostInfo)
}

func (p *connPool) report() {
	// report
	utils.GoWithRecover(func() {
		for {
			p.clientMux.Lock()
			log.DefaultLogger.Infof("[stream] [http] [connpool] pool = %s, available clients=%d, total clients=%d\n", p.host.AddressString(), len(p.availableClients), p.totalClientCount)
			p.clientMux.Unlock()
			time.Sleep(time.Second)
		}
	}, nil)
}

// types.StreamEventListener
// types.ConnectionEventListener
// types.StreamConnectionEventListener
type activeClient struct {
	pool               *connPool
	client             str.Client
	host               types.CreateConnectionData
	totalStream        uint64
	closeWithActiveReq bool
	closed             bool
	closeConn          bool
}

func newActiveClient(ctx context.Context, pool *connPool) (*activeClient, types.PoolFailureReason) {
	ac := &activeClient{
		pool: pool,
	}

	data := pool.host.CreateConnection(ctx)
	codecClient := pool.createStreamClient(ctx, data)
	codecClient.AddConnectionEventListener(ac)
	codecClient.SetStreamConnectionEventListener(ac)

	ac.client = codecClient
	ac.host = data

	if err := ac.client.Connect(); err != nil {
		return nil, types.ConnectionFailure
	}

	pool.host.HostStats().UpstreamConnectionTotal.Inc(1)
	pool.host.HostStats().UpstreamConnectionActive.Inc(1)
	pool.host.ClusterInfo().Stats().UpstreamConnectionTotal.Inc(1)
	pool.host.ClusterInfo().Stats().UpstreamConnectionActive.Inc(1)

	// bytes total adds all connections data together
	codecClient.SetConnectionCollector(pool.host.ClusterInfo().Stats().UpstreamBytesReadTotal, pool.host.ClusterInfo().Stats().UpstreamBytesWriteTotal)

	return ac, ""
}

// types.ConnectionEventListener
func (ac *activeClient) OnEvent(event types.ConnectionEvent) {
	ac.pool.onConnectionEvent(ac, event)
}

// types.StreamEventListener
func (ac *activeClient) OnDestroyStream() {
	if !ac.closed && ac.closeConn {
		ac.client.Close()
	}
	ac.pool.onStreamDestroy(ac)
}

func (ac *activeClient) OnResetStream(reason types.StreamResetReason) {
	ac.pool.onStreamReset(ac, reason)
	if reason == types.StreamLocalReset && !ac.closed {
		log.DefaultLogger.Debugf("[stream] [http] stream local reset, blow client away also, Connection = %d",
			ac.client.ConnID())
		ac.closeConn = true
	}
}

// types.StreamConnectionEventListener
func (ac *activeClient) OnGoAway() {
	ac.closeConn = true
}
