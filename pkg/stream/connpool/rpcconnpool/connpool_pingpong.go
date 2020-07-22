package connpool

import (
	"context"
	"sync/atomic"
	"time"

	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
)

type connpoolPingPong struct {
	*connpool
	idleClients map[api.Protocol][]*activeClientPingPong
}

// CheckAndInit init the connection pool
func (p *connpoolPingPong) CheckAndInit(ctx context.Context) bool {
	return true
}

// NewStream Create a client stream and call's by proxy
func (p *connpoolPingPong) NewStream(ctx context.Context, receiver types.StreamReceiveListener) (types.PoolFailureReason, types.Host, types.StreamSender) {
	host := p.Host()

	c, reason := p.GetActiveClient(ctx, getSubProtocol(ctx))
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
func (p *connpoolPingPong) GetActiveClient(ctx context.Context, subProtocol types.ProtocolName) (*activeClientPingPong, types.PoolFailureReason) {

	host := p.Host()
	if !host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
		return nil, types.Overflow
	}

	p.clientMux.Lock()

	n := len(p.idleClients[subProtocol])

	// max conns is 0 means no limit
	maxConns := host.ClusterInfo().ResourceManager().Connections().Max()
	// no available client
	var (
		c      *activeClientPingPong
		reason types.PoolFailureReason
	)

	if n == 0 { // nolint: nestif
		if maxConns == 0 || p.totalClientCount < maxConns {
			// connection not multiplex,
			// so we can concurrently build connections here
			p.clientMux.Unlock()
			c, reason = p.newActiveClient(ctx, subProtocol)
			if c != nil && reason == "" {
				p.totalClientCount++
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
		// Only refuse extra connection, keepalive-connection is closed by timeout
		usedConns := p.totalClientCount - uint64(n) + 1
		if maxConns != 0 && usedConns > host.ClusterInfo().ResourceManager().Connections().Max() {
			host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
			host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
			c, reason = nil, types.Overflow
			goto RET
		}

		c = p.idleClients[subProtocol][lastIdx]
		p.idleClients[subProtocol][lastIdx] = nil
		p.idleClients[subProtocol] = p.idleClients[subProtocol][:lastIdx]

		goto RET
	}

RET:

	if c != nil && reason == "" {
		host.HostStats().UpstreamRequestTotal.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestTotal.Inc(1)
	}

	return c, reason
}

func (p *connpoolPingPong) Close() {
	p.clientMux.Lock()
	defer p.clientMux.Unlock()
	atomic.StoreUint64(&p.destroyed, 1)

	for _, clients := range p.idleClients {
		for _, c := range clients {
			c.host.Connection.Close(api.NoFlush, api.LocalClose)
		}
	}
}

func (p *connpoolPingPong) Shutdown() {
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

// return client to pool
func (p *connpoolPingPong) putClientToPoolLocked(client *activeClientPingPong) {
	subProto := client.subProtocol

	if !client.closed {
		p.idleClients[subProto] = append(p.idleClients[subProto], client)
	}
}

// types.StreamEventListener
// types.ConnectionEventListener
// types.StreamConnectionEventListener
type activeClientPingPong struct {
	pool        *connpoolPingPong
	codecClient types.StreamClient
	host        types.CreateConnectionData

	closeWithActiveReq bool

	// -----http1 && ping pong mode
	closed          bool
	shouldCloseConn bool
	// -----http1 end

	// -----xprotocol start
	subProtocol types.ProtocolName
	keepAlive   *keepAliveListener
	state       uint32 // for async connection
	// -----xprotocol end
}

func (p *connpoolPingPong) newActiveClient(ctx context.Context, subProtocol api.Protocol) (*activeClientPingPong, types.PoolFailureReason) {
	// if pool is already destroyed, return
	if atomic.LoadUint64(&p.destroyed) == 1 {
		return nil, types.ConnectionFailure
	}

	ac := &activeClientPingPong{
		pool:        p,
		subProtocol: subProtocol,
		host:        p.Host().CreateConnection(ctx),
	}

	host := p.Host()
	connCtx := ctx

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
func (ac *activeClientPingPong) Close(err error) {
	if err != nil {
		// if pool is not using multiplex mode
		// this conn is not in the pool
		ac.host.Connection.Close(api.NoFlush, api.LocalClose)
		return
	}

	// xprotocol ping pong
	ac.host.Connection.Close(api.NoFlush, api.LocalClose)

	// return to pool
	ac.pool.clientMux.Lock()
	defer ac.pool.clientMux.Unlock()
	ac.pool.putClientToPoolLocked(ac)
}

// removeFromPool removes this client from connection pool
func (ac *activeClientPingPong) removeFromPool() {
	p := ac.pool
	subProtocol := ac.subProtocol
	p.clientMux.Lock()

	defer p.clientMux.Unlock()
	p.totalClientCount--
	for idx, c := range p.idleClients[subProtocol] {
		if c == ac {
			// remove this element
			lastIdx := len(p.idleClients[subProtocol]) - 1
			// 	1. swap this with the last
			p.idleClients[subProtocol][idx], p.idleClients[subProtocol][lastIdx] =
				p.idleClients[subProtocol][lastIdx], p.idleClients[subProtocol][idx]
			// 	2. set last to nil
			p.idleClients[subProtocol][lastIdx] = nil
			// 	3. remove the last
			p.idleClients[subProtocol] = p.idleClients[subProtocol][:lastIdx]
		}
	}
	ac.closed = true
}

// types.ConnectionEventListener
func (ac *activeClientPingPong) OnEvent(event api.ConnectionEvent) {
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

	switch {
	case event.IsClose():
		if p.protocol == protocol.Xprotocol {
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
		}

		// RemoteClose when read/write error
		// LocalClose when there is a panic
		// OnReadErrClose when read failed
		ac.removeFromPool()

	case event == api.ConnectTimeout:
		host.HostStats().UpstreamRequestTimeout.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestTimeout.Inc(1)
		ac.codecClient.Close()
	case event == api.ConnectFailed:
		host.HostStats().UpstreamConnectionConFail.Inc(1)
		host.ClusterInfo().Stats().UpstreamConnectionConFail.Inc(1)
	default:
		// do nothing
	}
}

// types.StreamEventListener
func (ac *activeClientPingPong) OnDestroyStream() {
	host := ac.pool.Host()
	host.HostStats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().Stats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().ResourceManager().Requests().Decrease()

	ac.Close(nil)
}

func (ac *activeClientPingPong) OnResetStream(reason types.StreamResetReason) {
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

	if reason == types.StreamLocalReset && !ac.closed {
		// for xprotocol ping pong
		log.DefaultLogger.Debugf("[stream] [http] stream local reset, blow codecClient away also, Connection = %d",
			ac.host.Connection.ID())
		ac.shouldCloseConn = true
	}
}

// types.StreamConnectionEventListener
func (ac *activeClientPingPong) OnGoAway() {
	ac.shouldCloseConn = true
}

// SetHeartBeater set the heart beat for an active client
func (ac *activeClientPingPong) SetHeartBeater(hb types.KeepAlive) {
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
