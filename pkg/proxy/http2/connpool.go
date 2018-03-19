package sofarpc

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"golang.org/x/net/http2"
)

// types.ConnectionPool
type connPool struct {
	primaryConn *activeClient
	drainConn   *activeClient
	host        types.Host
}

func (p *connPool) Protocol() types.Protocol {
	return protocol.Http2
}

func (p *connPool) AddDrainedCallback(cb func()) {}

func (p *connPool) DrainConnections() {}

func (p *connPool) NewStream(streamId uint32, responseDecoder types.StreamDecoder,
	cb types.PoolCallbacks) types.Cancellable {

}

func (p *connPool) Close() {}

type activeClient struct {
	pool        *connPool
	codecClient http2.ClientConn
	host        types.HostInfo
	totalStream uint64
}





