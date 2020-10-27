package xprotocol

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/pkg/buffer"
	"testing"
	"time"
)

func TestPingPong(t *testing.T) {
	var addr = "127.0.0.1:10086"
	go server.start(t, addr)
	defer server.stop(t)
	// wait for server to start
	time.Sleep(time.Second * 2)

	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyConfigUpStreamProtocol, string(protocol.Xprotocol))
	ctx = mosnctx.WithValue(ctx, types.ContextSubProtocol, "dubbo")

	cl := basicCluster(addr, []string{addr})
	host := cluster.NewSimpleHost(cl.Hosts[0], cluster.NewCluster(cl).Snapshot().ClusterInfo())

	p := connpool{
		protocol: protocol.Xprotocol,
		tlsHash:  &types.HashValue{},
	}
	p.host.Store(host)

	pMultiplex := NewPoolPingPong(&p)
	pInst := pMultiplex.(*poolPingPong)
	var xsList []*xStream
	for i := 0; i < 10; i++ {
		_, sender, failReason := pInst.NewStream(ctx, &receiver{})
		assert.Equal(t, types.PoolFailureReason(""), failReason)
		xs := sender.(*xStream)
		xs.direction = stream.ServerStream
		xsList = append(xsList, xs)
	}

	// destroy all streams
	// these connections should all go back go idleClients
	for i := 0; i < len(xsList); i++ {
		xsList[i].AppendHeaders(context.TODO(), &dubbo.Frame{Header: dubbo.Header{
			Magic:     []byte{1, 2},
			Direction: 0,
		}}, true)
	}

	assert.Equal(t, len(xsList), len(pInst.idleClients[api.Protocol("dubbo")]))
}

type receiver struct{}

func (r *receiver) OnReceive(ctx context.Context, headers api.HeaderMap,
	data buffer.IoBuffer, trailers api.HeaderMap) {
	fmt.Println("receive data")
}

func (r *receiver) OnDecodeError(ctx context.Context, err error,
	headers api.HeaderMap) {
	fmt.Println("decode error")
}
