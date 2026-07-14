package http2

import (
	"context"
	"mosn.io/pkg/variable"
	"sync"
	"sync/atomic"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	stream2 "mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

// for xprotocol
const (
	Init = iota
	Connecting
	Connected
)

// poolMultiplex a single pool is connections which can be reused in a single host
type poolMultiplex struct {
	*connPool

	clientMux              sync.Mutex
	activeClients          []sync.Map
	currentCheckAndInitIdx int64

	shutdown bool // pool is already shutdown
}

var (
	defaultMaxConn         = 1
	connNumberLimit uint64 = 65535 // port limit
)

func isValidMaxNum(maxConns uint64) bool {
	// xDS cluster if not limit max connection will recv:
	// max_connections:{value:4294967295}  max_pending_requests:{value:4294967295}  max_requests:{value:4294967295}  max_retries:{value:4294967295}
	// if not judge max, will oom
	return maxConns > 0 && maxConns < connNumberLimit
}

// SetDefaultMaxConnNumPerHostPortForMuxPool set the max connections for each host:port
// users could use this function or cluster threshold config to configure connection no.
func SetDefaultMaxConnNumPerHostPortForMuxPool(maxConns int) {
	if isValidMaxNum(uint64(maxConns)) {
		defaultMaxConn = maxConns
	}
}

// NewPoolMultiplex generates a multiplex conn pool
func NewPoolMultiplex(p *connPool) types.ConnectionPool {
	maxConns := p.Host().ClusterInfo().ResourceManager().Connections().Max()

	// the cluster threshold config has higher privilege than global config
	// if a valid number is provided, should use it
	if !isValidMaxNum(maxConns) {
		// override maxConns by default max conns
		maxConns = uint64(defaultMaxConn)
	}

	return &poolMultiplex{
		connPool:      p,
		activeClients: make([]sync.Map, maxConns),
	}
}

func (p *poolMultiplex) init(client *activeClientMultiplex, sub types.ProtocolName, index int) {
	utils.GoWithRecover(func() {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[stream] [http2] [connpool] init host %s", p.Host().AddressString())
		}

		p.clientMux.Lock()
		defer p.clientMux.Unlock()

		// if the pool is already shut down, do nothing directly return
		if p.shutdown {
			return
		}
		ctx := context.Background()
		client, _ := p.newActiveClient(ctx, sub)
		if client != nil {
			client.state = Connected
			client.indexInPool = index
			p.activeClients[index].Store(sub, client)
		} else {
			p.activeClients[index].Delete(sub)
		}

	}, nil)
}

// CheckAndInit init the connection pool
func (p *poolMultiplex) CheckAndInit(ctx context.Context) bool {
	var clientIdx int64 = 0 // most use cases, there will only be 1 connection
	if len(p.activeClients) > 1 {
		if clientIdx = getClientIDFromDownStreamCtx(ctx); clientIdx == invalidClientID {
			clientIdx = atomic.AddInt64(&p.currentCheckAndInitIdx, 1) % int64(len(p.activeClients))
			// set current client index to downstream context
			_ = variable.Set(ctx, types.VariableConnectionPoolIndex, clientIdx)
		}
	}

	var client *activeClientMultiplex
	subProtocol := p.Protocol()

	v, ok := p.activeClients[clientIdx].Load(p.Protocol())
	if !ok {
		fakeclient := &activeClientMultiplex{}
		fakeclient.state = Init
		v, _ := p.activeClients[clientIdx].LoadOrStore(subProtocol, fakeclient)
		client = v.(*activeClientMultiplex)
	} else {
		client = v.(*activeClientMultiplex)
	}

	if atomic.LoadUint32(&client.state) == Connected {
		return true
	}

	if atomic.CompareAndSwapUint32(&client.state, Init, Connecting) {
		p.init(client, subProtocol, int(clientIdx))
	}

	return false
}

// NewStream Create a client stream and call's by proxy
func (p *poolMultiplex) NewStream(ctx context.Context, receiver types.StreamReceiveListener) (types.Host, types.StreamSender, types.PoolFailureReason) {
	var (
		ok        bool
		clientIdx int64 = 0
	)

	if len(p.activeClients) > 1 {
		clientIdxInter, _ := variable.Get(ctx, types.VariableConnectionPoolIndex)
		if clientIdx, ok = clientIdxInter.(int64); !ok {
			// this client is not inited
			return p.Host(), nil, types.ConnectionFailure
		}
	}

	subProtocol := p.Protocol()

	client, _ := p.activeClients[clientIdx].Load(subProtocol)

	host := p.Host()
	if client == nil {
		return host, nil, types.ConnectionFailure
	}

	activeClient := client.(*activeClientMultiplex)
	if atomic.LoadUint32(&activeClient.state) != Connected {
		return host, nil, types.ConnectionFailure
	}

	if !host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
		return host, nil, types.Overflow
	}

	atomic.AddUint64(&activeClient.totalStream, 1)
	host.HostStats().UpstreamRequestTotal.Inc(1)
	host.ClusterInfo().Stats().UpstreamRequestTotal.Inc(1)

	var streamEncoder types.StreamSender
	// oneway
	if receiver == nil {
		streamEncoder = activeClient.codecClient.NewStream(ctx, nil)
	} else {
		streamEncoder = activeClient.codecClient.NewStream(ctx, receiver)
		streamEncoder.GetStream().AddEventListener(activeClient)
		host.HostStats().UpstreamRequestActive.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestActive.Inc(1)
		host.ClusterInfo().ResourceManager().Requests().Increase()
	}

	return host, streamEncoder, ""
}

// Shutdown stop the keepalive, so the connection will be idle after requests finished
func (p *poolMultiplex) Shutdown() {
	utils.GoWithRecover(func() {
		{
			p.clientMux.Lock()
			if p.shutdown {
				p.clientMux.Unlock()
				return
			}
			p.shutdown = true
			p.clientMux.Unlock()
		}

		for i := 0; i < len(p.activeClients); i++ {
			f := func(k, v interface{}) bool {
				ac, _ := v.(*activeClientMultiplex)
				if ac.keepAlive != nil {
					ac.keepAlive.keepAlive.Stop()
				}
				return true
			}
			p.activeClients[i].Range(f)
		}

	}, nil)
}

func (p *poolMultiplex) createStreamClient(context context.Context, connData types.CreateConnectionData) stream2.Client {
	return stream2.NewStreamClient(context, protocol.HTTP2, connData.Connection, connData.Host)
}

func (p *poolMultiplex) newActiveClient(ctx context.Context, subProtocol api.ProtocolName) (*activeClientMultiplex, types.PoolFailureReason) {
	ac := &activeClientMultiplex{
		subProtocol: subProtocol,
		pool:        p,
	}

	host := p.Host()
	data := host.CreateConnection(ctx)
	connCtx := ctx
	_ = variable.Set(ctx, types.VariableConnectionID, data.Connection.ID())
	codecClient := p.createStreamClient(connCtx, data)
	codecClient.AddConnectionEventListener(ac)
	codecClient.SetStreamConnectionEventListener(ac)

	ac.codecClient = codecClient
	ac.host = data

	// create keepalive
	newHttp2KeepAlive := NewKeepAlive(codecClient, subProtocol, time.Second)
	newHttp2KeepAlive.StartIdleTimeout()
	ac.keepAlive = &keepAliveListener{
		keepAlive: newHttp2KeepAlive,
	}
	ac.codecClient.AddConnectionEventListener(ac.keepAlive)

	if err := ac.codecClient.Connect(); err != nil {
		return nil, types.ConnectionFailure
	}

	// stats
	host.HostStats().UpstreamConnectionTotal.Inc(1)
	host.HostStats().UpstreamConnectionActive.Inc(1)
	host.ClusterInfo().Stats().UpstreamConnectionTotal.Inc(1)
	host.ClusterInfo().Stats().UpstreamConnectionActive.Inc(1)

	// bytes total adds all connections data together
	codecClient.SetConnectionCollector(host.ClusterInfo().Stats().UpstreamBytesReadTotal, host.ClusterInfo().Stats().UpstreamBytesWriteTotal)

	return ac, ""
}

func (p *poolMultiplex) Close() {
	for i := 0; i < len(p.activeClients); i++ {
		f := func(k, v interface{}) bool {
			ac, _ := v.(*activeClientMultiplex)
			if ac.codecClient != nil {
				ac.codecClient.Close()
			}
			return true
		}

		p.activeClients[i].Range(f)
	}
}

func (p *poolMultiplex) onConnectionEvent(ac *activeClientMultiplex, event api.ConnectionEvent) {
	host := p.Host()
	// event.ConnectFailure() contains types.ConnectTimeout and types.ConnectTimeout
	if event.IsClose() {
		host.HostStats().UpstreamConnectionClose.Inc(1)
		host.HostStats().UpstreamConnectionActive.Dec(1)

		host.ClusterInfo().Stats().UpstreamConnectionClose.Inc(1)
		host.ClusterInfo().Stats().UpstreamConnectionActive.Dec(1)

		switch event {
		case api.LocalClose:
			host.HostStats().UpstreamConnectionLocalClose.Inc(1)
			host.ClusterInfo().Stats().UpstreamConnectionLocalClose.Inc(1)

			if ac.closeWithActiveReq {
				host.HostStats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
				host.ClusterInfo().Stats().UpstreamConnectionLocalCloseWithActiveRequest.Inc(1)
			}

		case api.RemoteClose:
			host.HostStats().UpstreamConnectionRemoteClose.Inc(1)
			host.ClusterInfo().Stats().UpstreamConnectionRemoteClose.Inc(1)

			if ac.closeWithActiveReq {
				host.HostStats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)
				host.ClusterInfo().Stats().UpstreamConnectionRemoteCloseWithActiveRequest.Inc(1)

			}
		default:
			// do nothing
		}
		p.clientMux.Lock()
		p.activeClients[ac.indexInPool].Delete(ac.subProtocol)
		p.clientMux.Unlock()
	} else if event == api.ConnectTimeout {
		host.HostStats().UpstreamRequestTimeout.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestTimeout.Inc(1)
		ac.codecClient.Close()
	} else if event == api.ConnectFailed {
		host.HostStats().UpstreamConnectionConFail.Inc(1)
		host.ClusterInfo().Stats().UpstreamConnectionConFail.Inc(1)
	}
}


// types.StreamEventListener
// types.ConnectionEventListener
// types.StreamConnectionEventListener
// nolint: maligned
type activeClientMultiplex struct {
	closeWithActiveReq bool
	totalStream        uint64
	subProtocol        types.ProtocolName
	keepAlive          *keepAliveListener
	state              uint32 // for async connection
	pool               *poolMultiplex
	indexInPool        int
	codecClient        stream2.Client
	host               types.CreateConnectionData
}

// types.ConnectionEventListener
// nolint: dupl
func (ac *activeClientMultiplex) OnEvent(event api.ConnectionEvent) {
	ac.pool.onConnectionEvent(ac, event)
}

// types.StreamEventListener
func (ac *activeClientMultiplex) OnDestroyStream() {
	host := ac.pool.Host()
	host.HostStats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().Stats().UpstreamRequestActive.Dec(1)
	host.ClusterInfo().ResourceManager().Requests().Decrease()
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
func (ac *activeClientMultiplex) OnGoAway() {}

const invalidClientID = -1

func getClientIDFromDownStreamCtx(ctx context.Context) int64 {
	clientIdxInter, _ := variable.Get(ctx, types.VariableConnectionPoolIndex)
	clientIdx, ok := clientIdxInter.(int64)
	if !ok {
		return invalidClientID
	}
	return clientIdx
}

// keepAliveListener is a types.ConnectionEventListener
type keepAliveListener struct {
	keepAlive types.KeepAlive
	conn      api.Connection
}

// OnEvent impl types.ConnectionEventListener
func (l *keepAliveListener) OnEvent(event api.ConnectionEvent) {
	if event == api.OnReadTimeout && l.keepAlive != nil {
		l.keepAlive.SendKeepAlive()
	}
}