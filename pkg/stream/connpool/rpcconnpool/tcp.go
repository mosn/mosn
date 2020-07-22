package connpool

import (
	"context"
	"sync/atomic"
	"time"

	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
)

type connpoolTCP struct {
	*connpool
	idleClients map[uint64][]*activeClientTCP
}

// CheckAndInit init the connection pool
func (p *connpoolTCP) CheckAndInit(ctx context.Context) bool {
	return true
}

// NewStream Create a client stream and call's by proxy
func (p *connpoolTCP) NewStream(ctx context.Context, receiver types.StreamReceiveListener, downstreamConn api.Connection) (types.PoolFailureReason, types.Host, types.StreamSender) {
	host := p.Host()

	c, reason := p.GetActiveClient(ctx, getSubProtocol(ctx))
	c.downstreamConn = downstreamConn

	if reason != "" {
		return reason, host, nil
	}

	var streamSender = c.codecClient.NewStream(ctx, receiver)

	streamSender.GetStream().AddEventListener(c) // OnResetStream, OnDestroyStream

	// FIXME one way
	// is there any need to skip the metrics?
	if receiver == nil {
		return "", host, streamSender
	}

	host.HostStats().UpstreamRequestActive.Inc(1)
	host.ClusterInfo().Stats().UpstreamRequestActive.Inc(1)
	host.ClusterInfo().ResourceManager().Requests().Increase()

	return "", host, streamSender
}

// GetActiveClient get a avail client
// nolint: dupl
func (p *connpoolTCP) GetActiveClient(ctx context.Context, subProtocol types.ProtocolName) (*activeClientTCP, types.PoolFailureReason) {

	host := p.Host()
	if !host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
		return nil, types.Overflow
	}

	p.clientMux.Lock()
	connID := getConnID(ctx)

	if len(p.idleClients[connID]) > 0 {
		defer p.clientMux.Unlock()

		// the client was inited in the CheckAndInit function
		lastIdx := len(p.idleClients[connID]) - 1
		return p.idleClients[connID][lastIdx], ""
	}

	n := len(p.idleClients[connID])

	// max conns is 0 means no limit
	maxConns := host.ClusterInfo().ResourceManager().Connections().Max()
	// no available client
	var (
		c      *activeClientTCP
		reason types.PoolFailureReason
	)

	if n == 0 { // nolint: nestif
		if maxConns == 0 || p.totalClientCount < maxConns {
			defer p.clientMux.Unlock()
			c, reason = p.newActiveClientTCP(ctx, subProtocol)
			if c != nil && reason == "" {
				p.totalClientCount++

				// HTTP/2 && xprotocol
				// should put this conn to pool
				p.idleClients[connID] = append(p.idleClients[connID], c)
			}

			goto RET
		} else {
			p.clientMux.Unlock()

			host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
			host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
			c, reason = nil, types.Overflow

			goto RET
		}
	} else {
		defer p.clientMux.Unlock()

		var lastIdx = n - 1
		var reason types.PoolFailureReason
		c = p.idleClients[connID][lastIdx]
		if c == nil || atomic.LoadUint32(&c.goaway) == 1 {
			c, reason = p.newActiveClientTCP(ctx, subProtocol)
			if reason == "" && c != nil {
				p.idleClients[connID][lastIdx] = c
			}
		}

		goto RET
	}

RET:
	if c != nil && atomic.LoadUint32(&c.state) != Connected {
		return nil, types.ConnectionFailure
	}

	if c != nil && reason == "" {
		atomic.AddUint64(&c.totalStream, 1)
		host.HostStats().UpstreamRequestTotal.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestTotal.Inc(1)
	}

	return c, reason
}

func (p *connpoolTCP) Close() {
	p.clientMux.Lock()
	defer p.clientMux.Unlock()
	atomic.StoreUint64(&p.destroyed, 1)

	for _, clients := range p.idleClients {
		for _, c := range clients {
			c.host.Connection.Close(api.NoFlush, api.LocalClose)
		}
	}
}

func (p *connpoolTCP) Shutdown() {
	p.clientMux.Lock()
	defer p.clientMux.Unlock()

	for _, clients := range p.idleClients {
		for _, c := range clients {
			c.OnGoAway()
			if c.keepAlive != nil {
				c.keepAlive.keepAlive.Stop()
			}
		}
	}
}

// types.StreamEventListener
// types.ConnectionEventListener
// types.StreamConnectionEventListener
// nolint: maligned
type activeClientTCP struct {
	closeWithActiveReq bool
	totalStream        uint64
	connID             uint64

	// -----http2 start
	goaway uint32
	// -----http2 end

	// -----xprotocol start
	subProtocol types.ProtocolName
	keepAlive   *keepAliveListener
	state       uint32 // for async connection
	// -----xprotocol end

	pool           *connpoolTCP
	codecClient    types.StreamClient
	host           types.CreateConnectionData
	downstreamConn api.Connection
}

func (p *connpoolTCP) newActiveClientTCP(ctx context.Context, subProtocol api.Protocol) (*activeClientTCP, types.PoolFailureReason) {
	// if pool is already destroyed, return
	if atomic.LoadUint64(&p.destroyed) == 1 {
		return nil, types.ConnectionFailure
	}

	connID := getConnID(ctx)
	ac := &activeClientTCP{
		subProtocol: subProtocol,
		connID:      connID,
		pool:        p,
		host:        p.Host().CreateConnection(ctx),
	}

	host := p.Host()

	connCtx := mosnctx.WithValue(ctx, types.ContextKeyConnectionID, ac.host.Connection.ID())

	if len(subProtocol) > 0 {
		connCtx = mosnctx.WithValue(ctx, types.ContextSubProtocol, string(subProtocol))
	}

	ac.host.Connection.AddConnectionEventListener(ac)

	// first connect to dest addr, then create stream client
	if err := ac.host.Connection.Connect(); err != nil {
		return nil, types.ConnectionFailure
	}

	////////// codec client
	codecClient := stream.NewStreamClient(connCtx, p.protocol, ac.host.Connection, host)
	codecClient.SetStreamConnectionEventListener(ac) // ac.OnGoAway
	ac.codecClient = codecClient
	// bytes total adds all connections data together
	codecClient.SetConnectionCollector(host.ClusterInfo().Stats().UpstreamBytesReadTotal, host.ClusterInfo().Stats().UpstreamBytesWriteTotal)

	if subProtocol != "" {
		// Add Keep Alive
		// protocol is from onNewDetectStream
		// check heartbeat enable, hack: judge trigger result of Heartbeater
		proto := xprotocol.GetProtocol(subProtocol)
		if heartbeater, ok := proto.(xprotocol.Heartbeater); ok && heartbeater.Trigger(0) != nil {
			// create keepalive
			rpcKeepAlive := NewKeepAlive(ac.codecClient, subProtocol, time.Second, 6)
			rpcKeepAlive.StartIdleTimeout()

			ac.SetHeartBeater(rpcKeepAlive)
		}
	}
	////////// codec client

	atomic.StoreUint32(&ac.state, Connected)

	// stats
	host.HostStats().UpstreamConnectionTotal.Inc(1)
	host.HostStats().UpstreamConnectionActive.Inc(1)
	host.ClusterInfo().Stats().UpstreamConnectionTotal.Inc(1)
	host.ClusterInfo().Stats().UpstreamConnectionActive.Inc(1)

	return ac, ""
}

// Close return this client back to pool
func (ac *activeClientTCP) Close(err error) {
	if err != nil {
		ac.removeFromPool()
		ac.host.Connection.Close(api.NoFlush, api.LocalClose)
		return
	}
}

// removeFromPool removes this client from connection pool
func (ac *activeClientTCP) removeFromPool() {
	p := ac.pool
	p.clientMux.Lock()

	defer p.clientMux.Unlock()
	p.totalClientCount--
	connID := ac.connID
	for idx, c := range p.idleClients[connID] {
		if c == ac {
			// remove this element
			lastIdx := len(p.idleClients[connID]) - 1
			// 	1. swap this with the last
			p.idleClients[connID][idx], p.idleClients[connID][lastIdx] =
				p.idleClients[connID][lastIdx], p.idleClients[connID][idx]
			// 	2. set last to nil
			p.idleClients[connID][lastIdx] = nil
			// 	3. remove the last
			p.idleClients[connID] = p.idleClients[connID][:lastIdx]
		}
	}
}

// types.ConnectionEventListener
// nolint: dupl
func (ac *activeClientTCP) OnEvent(event api.ConnectionEvent) {
	p := ac.pool

	host := p.Host()
	// all protocol should report the following metrics
	if ac.closeWithActiveReq {
		if event == api.LocalClose {
			host.HostStats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
			host.ClusterInfo().Stats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
		} else if event == api.RemoteClose {
			host.HostStats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)
			host.ClusterInfo().Stats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)
		}
	}

	var needCloseDownStream = false
	switch {
	case event.IsClose():
		host.HostStats().UpstreamConnectionClose.Inc(1)
		host.HostStats().UpstreamConnectionActive.Dec(1)
		host.ClusterInfo().Stats().UpstreamConnectionClose.Inc(1)
		host.ClusterInfo().Stats().UpstreamConnectionActive.Dec(1)

		switch event { // nolint: exhaustive
		case api.LocalClose:
			host.HostStats().UpstreamConnectionLocalClose.Inc(1)
			host.ClusterInfo().Stats().UpstreamConnectionLocalClose.Inc(1)
		case api.RemoteClose:
			host.HostStats().UpstreamConnectionRemoteClose.Inc(1)
			host.ClusterInfo().Stats().UpstreamConnectionRemoteClose.Inc(1)
		default:
			// do nothing
		}

		// RemoteClose when read/write error
		// LocalClose when there is a panic
		// OnReadErrClose when read failed
		ac.removeFromPool()
		needCloseDownStream = true

	case event == api.ConnectTimeout:
		host.HostStats().UpstreamRequestTimeout.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestTimeout.Inc(1)
		ac.codecClient.Close()
		needCloseDownStream = true
	case event == api.ConnectFailed:
		host.HostStats().UpstreamConnectionConFail.Inc(1)
		host.ClusterInfo().Stats().UpstreamConnectionConFail.Inc(1)
		needCloseDownStream = true
	default:
		// do nothing
	}

	if needCloseDownStream && ac.downstreamConn != nil {
		ac.downstreamConn.Close(api.NoFlush, api.LocalClose)
	}
}

// types.StreamEventListener
func (ac *activeClientTCP) OnDestroyStream() {
	host := ac.pool.Host()
	host.HostStats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().Stats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().ResourceManager().Requests().Decrease()

	ac.Close(nil)
}

func (ac *activeClientTCP) OnResetStream(reason types.StreamResetReason) {
	host := ac.pool.Host()
	switch reason {
	case types.StreamConnectionTermination, types.StreamConnectionFailed:
		host.HostStats().UpstreamRequestFailureEject.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestFailureEject.Inc(1)
		ac.closeWithActiveReq = true
	case types.StreamLocalReset:
		host.HostStats().UpstreamRequestLocalReset.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestLocalReset.Inc(1)
	case types.StreamRemoteReset:
		host.HostStats().UpstreamRequestRemoteReset.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestRemoteReset.Inc(1)
	}
}

// types.StreamConnectionEventListener
func (ac *activeClientTCP) OnGoAway() {
	atomic.StoreUint32(&ac.goaway, 1)
}

// SetHeartBeater set the heart beat for an active client
func (ac *activeClientTCP) SetHeartBeater(hb types.KeepAlive) {
	// clear the previous keepAlive
	if ac.keepAlive != nil && ac.keepAlive.keepAlive != nil {
		ac.keepAlive.keepAlive.Stop()
		ac.keepAlive = nil
	}

	ac.keepAlive = &keepAliveListener{
		keepAlive: hb,
		conn:      ac.host.Connection,
	}

	ac.host.Connection.AddConnectionEventListener(ac.keepAlive)
}

func getConnID(ctx context.Context) uint64 {
	if ctx != nil {
		if val := mosnctx.Get(ctx, types.ContextKeyConnectionID); val != nil {
			if code, ok := val.(uint64); ok {
				return code
			}
		}
	}
	return 0
}
