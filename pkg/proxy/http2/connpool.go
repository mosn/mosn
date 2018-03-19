package sofarpc

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/proxy"
	"sync"
)

// types.ConnectionPool
type connPool struct {
	primaryClient  *activeClient
	drainingClient *activeClient
	mux            sync.Mutex
	host           types.Host
}

func NewConnPool(host types.Host) types.ConnectionPool {
	return &connPool{
		host: host,
	}
}

func (p *connPool) Protocol() types.Protocol {
	return protocol.SofaRpc
}

func (p *connPool) AddDrainedCallback(cb func()) {}

func (p *connPool) DrainConnections() {}

func (p *connPool) NewStream(streamId uint32, responseDecoder types.StreamDecoder,
	cb types.PoolCallbacks) types.Cancellable {
	p.mux.Lock()

	if p.primaryClient == nil {
		p.primaryClient = newActiveClient(p)
	}
	p.mux.Unlock()

	if !p.host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		cb.OnPoolFailure(streamId, types.Overflow, nil)
	} else {
		// todo: update host stats
		p.primaryClient.totalStream++
		p.host.ClusterInfo().ResourceManager().Requests().Increase()
		streamEncoder := p.primaryClient.codecClient.NewStream(streamId, responseDecoder)
		cb.OnPoolReady(streamId, streamEncoder, p.host)
	}

	return nil
}

func (p *connPool) Close() {
	p.mux.Lock()
	defer p.mux.Unlock()

	if p.primaryClient != nil {
		p.primaryClient.codecClient.Close()
	}
}

func (p *connPool) onConnectionEvent(client *activeClient, event types.ConnectionEvent) {
	if event.IsClose() {
		// todo: update host stats

		p.mux.Lock()
		defer p.mux.Unlock()

		if p.primaryClient == client {
			p.primaryClient = nil
		}
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

func (p *connPool) onGoAway(client *activeClient) {
	// todo: update host stats

	p.mux.Lock()
	defer p.mux.Unlock()

	if p.primaryClient == client {
		p.movePrimaryToDraining()
	}
}

func (p *connPool) createCodecClient(connData types.CreateConnectionData) proxy.CodecClient {
	return proxy.NewCodecClient(protocol.Http2, connData.Connection, connData.HostInfo)
}

func (p *connPool) movePrimaryToDraining() {
	if p.drainingClient != nil {
		p.drainingClient.codecClient.Close()
	}

	if p.primaryClient.codecClient.ActiveRequestsNum() == 0 {
		p.primaryClient.codecClient.Close()
	} else {
		p.drainingClient = p.primaryClient
		p.primaryClient = nil
	}
}

// proxy.CodecClientCallbacks
// types.ConnectionCallbacks
// types.StreamConnectionCallbacks
type activeClient struct {
	pool        *connPool
	codecClient proxy.CodecClient
	host        types.HostInfo
	totalStream uint64
}

func newActiveClient(pool *connPool) *activeClient {
	ac := &activeClient{
		pool: pool,
	}

	data := pool.host.CreateConnection()
	codecClient := pool.createCodecClient(data)
	codecClient.AddConnectionCallbacks(ac)
	codecClient.SetCodecClientCallbacks(ac)
	codecClient.SetCodecConnectionCallbacks(ac)

	ac.codecClient = codecClient
	ac.host = data.HostInfo

	data.Connection.Connect(false)

	return ac
}

func (ac *activeClient) OnEvent(event types.ConnectionEvent) {
	ac.pool.onConnectionEvent(ac, event)
}

func (ac *activeClient) OnAboveWriteBufferHighWatermark() {}

func (ac *activeClient) OnBelowWriteBufferLowWatermark() {}

func (ac *activeClient) OnStreamDestroy() {
	ac.pool.onStreamDestroy(ac)
}

func (ac *activeClient) OnStreamReset(reason types.StreamResetReason) {
	ac.pool.onStreamReset(ac, reason)
}

func (ac *activeClient) OnGoAway() {
	ac.pool.onGoAway(ac)
}
