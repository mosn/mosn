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
package msgconnpool

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/upstream/cluster"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"
)

// for reconnect
const (
	notConnecting = iota
	connecting
)

// types.ConnectionPool
type connpool struct {
	idleClients []*activeClient

	host      types.Host
	clientMux sync.Mutex

	autoReconnectWhenClose bool
	heartBeatCreator       func() KeepAlive
	reconnTryTimes         int
	readFilters            []api.ReadFilter

	destroyed uint64
}

func (p *connpool) Host() types.Host {
	return p.host
}

// GetActiveClient get a avail client
func (p *connpool) GetActiveClient(ctx context.Context) (*activeClient, types.PoolFailureReason) {
	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	n := len(p.idleClients)

	// no available client
	var (
		c      *activeClient
		reason types.PoolFailureReason
	)

	if n == 0 {
		c, reason = p.newActiveClientLocked(ctx)
		// if the reason if connection failure
		// will automatic reconnect, so we can save the active client
		// to avoid concurrent reconnect
		if c != nil {
			// should put this conn to pool
			p.idleClients = append(p.idleClients, c)
		}

		return c, reason
	} else {

		var lastIdx = n - 1
		var reason types.PoolFailureReason
		c = p.idleClients[lastIdx]
		if c == nil {
			c, reason = p.newActiveClientLocked(ctx)
			if reason == "" && c != nil {
				p.idleClients[lastIdx] = c
			}
		}

		return c, reason
	}
}

// Destroy the pool
func (p *connpool) Destroy() {
	atomic.StoreUint64(&p.destroyed, 1)

	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	for _, c := range p.idleClients {
		c.host.Connection.Close(api.NoFlush, api.LocalClose)
	}
}

// types.StreamEventListener
// types.ConnectionEventListener
// types.StreamConnectionEventListener
type activeClient struct {
	pool *connpool
	host *types.CreateConnectionData

	keepAlive *keepAliveListener

	reconnectState uint64 // for reconnect, connecting or notConnecting
}

func (p *connpool) newActiveClientLocked(ctx context.Context) (*activeClient, types.PoolFailureReason) {
	// if pool is already destroyed, return
	if atomic.LoadUint64(&p.destroyed) == 1 {
		return nil, types.ConnectionFailure
	}

	createConnData := p.Host().CreateConnection(ctx)
	ac := &activeClient{
		pool: p,
		host: &createConnData,
	}

	ac.host.Connection.AddConnectionEventListener(ac)

	// first connect to dest addr, then create stream client
	if err := ac.host.Connection.Connect(); err != nil {
		if p.autoReconnectWhenClose && atomic.LoadUint64(&p.destroyed) == 0 {
			// auto reconnect when the first connect failed
			log.DefaultLogger.Warnf("[connpool] reconnect due to first connect failed %v", ac.host.Host.AddressString())
			ac.Reconnect()
		}

		return ac, types.ConnectionFailure
	} else {
		if atomic.LoadUint64(&p.destroyed) == 1 {
			// if destroyed, close the conn
			ac.host.Connection.Close(api.NoFlush, api.LocalClose)
			return ac, types.ConnectionFailure
		}

		for _, rf := range ac.pool.readFilters {
			ac.host.Connection.FilterManager().AddReadFilter(rf)
		}
	}

	// if user use connection pool without codec
	// they can use the connection returned from activeClient.Conn()
	if p.heartBeatCreator != nil {
		ac.SetHeartBeater(p.heartBeatCreator())
	}

	return ac, ""
}

// Reconnect triggers connection to reconnect
func (ac *activeClient) Reconnect() {
	if !atomic.CompareAndSwapUint64(&ac.reconnectState, notConnecting, connecting) {
		return
	}

	if atomic.LoadUint64(&ac.pool.destroyed) == 1 {
		return
	}

	utils.GoWithRecover(func() {
		defer atomic.CompareAndSwapUint64(&ac.reconnectState, connecting, notConnecting)
		var (
			err error
			i   int
		)

		// close previous conn
		ac.host.Connection.Close(api.NoFlush, api.RemoteClose)

		var (
			backoffArr = []time.Duration{
				time.Second,
				time.Second * 2,
				time.Second * 5,
				time.Second * 10,
			}
			backoffIdx = 0
		)

		for ; i < ac.pool.reconnTryTimes; i++ {
			if atomic.LoadUint64(&ac.pool.destroyed) == 1 {
				// if pool was exited, then stop
				return
			}
			// build new conn
			// must create this new conn, the same conn can only be connected once
			createConnData := ac.pool.Host().CreateConnection(context.Background())


			// connect the new connection
			err = createConnData.Connection.Connect()
			if err != nil {

				// backoff logic
				if backoffIdx >= len(backoffArr) {
					backoffIdx = len(backoffArr) - 1
				}

				log.DefaultLogger.Warnf("[connpool] reconnect failed %v times, host %v, ac : %p, backoff : %v",
					i+1, ac.host.Host.AddressString(), ac, backoffArr[backoffIdx])

				time.Sleep(backoffArr[backoffIdx])
				backoffIdx++
				continue
			}

			// atomic store, avoid partial write
			atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&ac.host)), unsafe.Pointer(&createConnData))

			log.DefaultLogger.Infof("[connpool] reconnect succeed after %v tries, host %v, ac %p", i+1, ac.host.Host.AddressString(), ac)

			// if pool was destroyed, but connection was connected
			// we need to close it
			if atomic.LoadUint64(&ac.pool.destroyed) == 1 {
				ac.host.Connection.Close(api.NoFlush, api.LocalClose)
				return
			}

			// ====== event listeners
			ac.host.Connection.AddConnectionEventListener(ac)
			if ac.keepAlive != nil {
				ac.host.Connection.AddConnectionEventListener(ac.keepAlive)
			}

			for _, rf := range ac.pool.readFilters {
				ac.host.Connection.FilterManager().AddReadFilter(rf)
			}

			// set the new heartbeat
			ac.SetHeartBeater(ac.pool.heartBeatCreator())

			// ====== event listeners
			// new conn should have read filters

			break
		}
	}, func(r interface{}) {
		log.DefaultLogger.Warnf("[connpool] reconnect failed, %v, host: %v", r, ac.host.Host.AddressString())
	})

}

// removeFromPool removes this client from connection pool
func (ac *activeClient) removeFromPool() {
	p := ac.pool
	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	for idx, c := range p.idleClients {
		if c == ac {
			// remove this element
			lastIdx := len(p.idleClients) - 1
			// 	1. swap this with the last
			p.idleClients[idx], p.idleClients[lastIdx] =
				p.idleClients[lastIdx], p.idleClients[idx]
			// 	2. set last to nil
			p.idleClients[lastIdx] = nil
			// 	3. remove the last
			p.idleClients = p.idleClients[:lastIdx]
		}
	}
}

// types.ConnectionEventListener
func (ac *activeClient) OnEvent(event api.ConnectionEvent) {
	p := ac.pool

	//  all close event:
	//  ce == LocalClose || ce == RemoteClose ||
	//	ce == OnReadErrClose || ce == OnWriteErrClose || ce == OnWriteTimeout
	switch event {
	case api.OnReadErrClose, api.OnWriteErrClose, api.RemoteClose, api.OnWriteTimeout, api.LocalClose:
		// RemoteClose when read/write error
		// LocalClose when there is a panic
		// OnReadErrClose when read failed
		log.DefaultLogger.Warnf("[connpool] reconnect after conn close, event : %v,  host : %v", event, ac.host.Host.AddressString())
		goto RECONN

	case api.ConnectTimeout:
		log.DefaultLogger.Warnf("[connpool] reconnect after connect timeout, host : %v", ac.host.Host.AddressString())
		goto RECONN
	case api.ConnectFailed:
		log.DefaultLogger.Warnf("[connpool] reconnect after connect failed, host : %v", ac.host.Host.AddressString())
		goto RECONN
	}

RECONN:
	if p.autoReconnectWhenClose && atomic.LoadUint64(&p.destroyed) == 0 {
		// auto reconnect when close
		ac.Reconnect()
	} else {
		ac.removeFromPool()
	}
}

// SetHeartBeater set the heart beat for an active client
func (ac *activeClient) SetHeartBeater(hb KeepAlive) {
	// clear the previous keepAlive
	if ac.keepAlive != nil && ac.keepAlive.keepAlive != nil {
		ac.keepAlive.keepAlive.Stop()
		ac.keepAlive = nil
	}

	ac.keepAlive = &keepAliveListener{
		keepAlive: hb,
		conn:      ac.host.Connection,
	}

	// this should be equal to
	// ac.codecClient.AddConnectionEventListener(ac.keepAlive)
	ac.host.Connection.AddConnectionEventListener(ac.keepAlive)
}

// keepAliveListener is a types.ConnectionEventListener
type keepAliveListener struct {
	keepAlive KeepAlive
	conn      api.Connection
}

// OnEvent impl types.ConnectionEventListener
func (l *keepAliveListener) OnEvent(event api.ConnectionEvent) {
	if event == api.OnReadTimeout && l.keepAlive != nil {
		heartbeatFailCreator := func() {
			l.conn.Close(api.NoFlush, api.LocalClose)
		}

		// TODO, whether error should be handled
		l.conn.Write(buffer.NewIoBufferBytes(l.keepAlive.GetKeepAliveData(heartbeatFailCreator)))
	}
}

type KeepAlive interface {
	Stop()

	GetKeepAliveData(failCallback func()) []byte
}

type Connection interface {
	Write(buf ...buffer.IoBuffer) error
	Destroy()
	Available() bool
}

// NewConn returns a simplified connpool
func NewConn(hostAddr string, reconnectTryTimes int, heartBeatCreator func() KeepAlive, readFilters []api.ReadFilter, autoReconnectWhenClose bool) Connection {
	// use host addr as cluster name, for the count of metrics
	cl := basicCluster(hostAddr, []string{hostAddr})
	host := cluster.NewSimpleHost(cl.Hosts[0], cluster.NewCluster(cl).Snapshot().ClusterInfo())

	// if user configure this to -1, then retry is unlimited
	if reconnectTryTimes == -1 {
		reconnectTryTimes = math.MaxInt32
	}

	p := &connpool{
		idleClients: make([]*activeClient, 0, 1),
		host:        host,

		heartBeatCreator:       heartBeatCreator,
		autoReconnectWhenClose: autoReconnectWhenClose,
		reconnTryTimes:         reconnectTryTimes,
		readFilters:            readFilters,
	}

	// trigger the client generation
	p.GetActiveClient(context.Background())

	return p
}

// write to client
func (p *connpool) Write(buf ...buffer.IoBuffer) error {
	cli, reason := p.GetActiveClient(context.Background())
	if reason != "" {
		return errors.New("get client failed" + string(reason))
	}

	// use atomic to load pointer, avoid partial pointer read
	h := (*types.CreateConnectionData)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cli.host))))

	err := h.Connection.Write(buf...)
	if err != nil {
		cli.Reconnect()
	}

	return err
}

// Available current available to send request
// WARNING, this api is only for msg
// WARNING, dont use it any other scene
func (p *connpool) Available() bool {
	// if pool was destroyed
	if atomic.LoadUint64(&p.destroyed) == 1 {
		return false
	}

	// if there is no client
	lastIdx := len(p.idleClients) - 1
	if lastIdx < 0 {
		return false
	}

	// if we are reconnecting
	if atomic.LoadUint64(&p.idleClients[lastIdx].reconnectState) == connecting {
		return false
	}

	return true
}

func basicCluster(name string, hosts []string) v2.Cluster {
	var vhosts []v2.Host
	for _, addr := range hosts {
		vhosts = append(vhosts, v2.Host{
			HostConfig: v2.HostConfig{
				Address: addr,
			},
		})
	}
	return v2.Cluster{
		Name:                 name,
		ClusterType:          v2.SIMPLE_CLUSTER,
		LbType:               v2.LB_ROUNDROBIN,
		ConnBufferLimitBytes: 16 * 1026,
		Hosts:                vhosts,
	}
}
