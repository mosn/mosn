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
	"mosn.io/pkg/utils"
)

type connpoolMultiplex struct {
	*connpool
	idleClients map[api.Protocol][]*activeClientMultiplex
}

// CheckAndInit init the connection pool
func (p *connpoolMultiplex) CheckAndInit(ctx context.Context) bool {
	subProtocol := getSubProtocol(ctx)

	var client *activeClientMultiplex

	// async connect only support multiplex mode !!!
	p.clientMux.Lock()
	{
		if len(p.idleClients[subProtocol]) == 0 {
			p.idleClients[subProtocol] = []*activeClientMultiplex{{state: Init}} // fake client
		}

		clients := p.idleClients[subProtocol]
		lastIdx := len(clients) - 1
		client = clients[lastIdx]
	}
	p.clientMux.Unlock()

	if atomic.LoadUint32(&client.state) == Connected {
		return true
	}

	// asynchronously connect to host
	// if there is a bad host, directly connect it may hang our request
	if atomic.CompareAndSwapUint32(&client.state, Init, Connecting) {
		utils.GoWithRecover(func() {
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[stream] [sofarpc] [connpool] init host %s", p.Host().AddressString())
			}

			p.clientMux.Lock()
			defer p.clientMux.Unlock()
			lastIdx := len(p.idleClients[subProtocol]) - 1
			client, _ := p.newActiveClientMultiplex(context.Background(), subProtocol)
			if client != nil {
				client.state = Connected
				p.idleClients[subProtocol][lastIdx] = client
			} else {
				delete(p.idleClients, subProtocol)
			}
		}, nil)
	}

	return false
}

// NewStream Create a client stream and call's by proxy
func (p *connpoolMultiplex) NewStream(ctx context.Context, receiver types.StreamReceiveListener) (types.PoolFailureReason, types.Host, types.StreamSender) {
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
func (p *connpoolMultiplex) GetActiveClient(ctx context.Context, subProtocol types.ProtocolName) (*activeClientMultiplex, types.PoolFailureReason) {

	host := p.Host()
	if !host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
		return nil, types.Overflow
	}

	p.clientMux.Lock()

	if len(p.idleClients[subProtocol]) > 0 {
		defer p.clientMux.Unlock()

		// the client was inited in the CheckAndInit function
		lastIdx := len(p.idleClients[subProtocol]) - 1
		return p.idleClients[subProtocol][lastIdx], ""
	}

	n := len(p.idleClients[subProtocol])

	// max conns is 0 means no limit
	maxConns := host.ClusterInfo().ResourceManager().Connections().Max()
	// no available client
	var (
		c      *activeClientMultiplex
		reason types.PoolFailureReason
	)

	if n == 0 { // nolint: nestif
		if maxConns == 0 || p.totalClientCount < maxConns {
			defer p.clientMux.Unlock()
			c, reason = p.newActiveClientMultiplex(ctx, subProtocol)
			if c != nil && reason == "" {
				p.totalClientCount++

				// HTTP/2 && xprotocol
				// should put this conn to pool
				p.idleClients[subProtocol] = append(p.idleClients[subProtocol], c)
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
		c = p.idleClients[subProtocol][lastIdx]
		if c == nil || atomic.LoadUint32(&c.goaway) == 1 {
			c, reason = p.newActiveClientMultiplex(ctx, subProtocol)
			if reason == "" && c != nil {
				p.idleClients[subProtocol][lastIdx] = c
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

func (p *connpoolMultiplex) Close() {
	p.clientMux.Lock()
	defer p.clientMux.Unlock()
	atomic.StoreUint64(&p.destroyed, 1)

	for _, clients := range p.idleClients {
		for _, c := range clients {
			c.host.Connection.Close(api.NoFlush, api.LocalClose)
		}
	}
}

func (p *connpoolMultiplex) Shutdown() {
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
func (p *connpoolMultiplex) putClientToPoolLocked(_ *activeClientPingPong) {}

// types.StreamEventListener
// types.ConnectionEventListener
// types.StreamConnectionEventListener
type activeClientMultiplex struct {
	pool        *connpoolMultiplex
	codecClient types.StreamClient
	host        types.CreateConnectionData

	totalStream        uint64
	closeWithActiveReq bool

	// -----http2 start
	goaway uint32
	// -----http2 end

	// -----xprotocol start
	subProtocol types.ProtocolName
	keepAlive   *keepAliveListener
	state       uint32 // for async connection
	// -----xprotocol end
}

func (p *connpoolMultiplex) newActiveClientMultiplex(ctx context.Context, subProtocol api.Protocol) (*activeClientMultiplex, types.PoolFailureReason) {
	// if pool is already destroyed, return
	if atomic.LoadUint64(&p.destroyed) == 1 {
		return nil, types.ConnectionFailure
	}

	ac := &activeClientMultiplex{
		pool:        p,
		subProtocol: subProtocol,
		host:        p.Host().CreateConnection(ctx),
	}

	host := p.Host()
	connCtx := ctx

	connCtx = mosnctx.WithValue(ctx, types.ContextKeyConnectionID, ac.host.Connection.ID())

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
func (ac *activeClientMultiplex) Close(err error) {
	if err != nil {
		ac.removeFromPool()
		ac.host.Connection.Close(api.NoFlush, api.LocalClose)
		return
	}
}

// removeFromPool removes this client from connection pool
func (ac *activeClientMultiplex) removeFromPool() {
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
}

// types.ConnectionEventListener
func (ac *activeClientMultiplex) OnEvent(event api.ConnectionEvent) {
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
func (ac *activeClientMultiplex) OnDestroyStream() {
	host := ac.pool.Host()
	host.HostStats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().Stats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().ResourceManager().Requests().Decrease()

	ac.Close(nil)
}

func (ac *activeClientMultiplex) OnResetStream(reason types.StreamResetReason) {
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
func (ac *activeClientMultiplex) OnGoAway() {
	atomic.StoreUint32(&ac.goaway, 1)
}

// SetHeartBeater set the heart beat for an active client
func (ac *activeClientMultiplex) SetHeartBeater(hb types.KeepAlive) {
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
