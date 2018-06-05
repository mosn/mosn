package sofarpc

import (
	"context"
	"errors"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
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

func (p *connPool) Host() types.Host {
	return p.host
}

func (p *connPool) DrainConnections() {}

func (p *connPool) InitActiveClient(context context.Context) error {
	if p.activeClient == nil {
		ac := newActiveClient(context, p)
		if ac == nil {
			return errors.New("Init Active Error")
		} else {
			p.activeClient = ac
		}
	}

	return nil
}

func (p *connPool) NewStream(context context.Context, streamId string,
	responseDecoder types.StreamDecoder, cb types.PoolEventListener) types.Cancellable {

	if !p.host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		cb.OnPoolFailure(streamId, types.Overflow, nil)
	} else {
		// todo: update host stats
		p.activeClient.totalStream++
		p.host.ClusterInfo().ResourceManager().Requests().Increase()
		streamEncoder := p.activeClient.codecClient.NewStream(streamId, responseDecoder)
		cb.OnPoolReady(streamId, streamEncoder, p.host)
	}

	return nil
}

func (p *connPool) Close() {
	if p.activeClient != nil {
		p.activeClient.codecClient.Close()
	}
}

func (p *connPool) onConnectionEvent(client *activeClient, event types.ConnectionEvent) {
	if event.IsClose()|| event == types.ConnectFailed {

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

// return nil if error occurs
func newActiveClient(context context.Context, pool *connPool) *activeClient {
	ac := &activeClient{
		pool: pool,
	}

	data := pool.host.CreateConnection(context)
	if err := data.Connection.Connect(true); err != nil {
		log.DefaultLogger.Errorf("Create Active Client Error,Remote Address is: %s, Err is %+v",
			pool.host.AddressString(),err)
		return nil
	}

	codecClient := pool.createCodecClient(context, data)
	codecClient.AddConnectionCallbacks(ac)
	codecClient.SetCodecClientCallbacks(ac)
	codecClient.SetCodecConnectionCallbacks(ac)

	ac.codecClient = codecClient
	ac.host = data.HostInfo
	log.DefaultLogger.Debugf("new client create, codecClient=%+v, host=%+v,connection data =%+v", codecClient, data.HostInfo, data)

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

func (ac *activeClient) OnGoAway() {}
