package xprotocol

import (
	"context"
	"github.com/stretchr/testify/assert"
	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"sync"
	"testing"
	"time"
)

const testClientNum = 10

func TestConnpoolMultiplexCheckAndInit(t *testing.T) {
	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyConfigUpStreamProtocol, string(protocol.Xprotocol))
	ctx = mosnctx.WithValue(ctx, types.ContextSubProtocol, "dubbo")
	ctxNew := mosnctx.Clone(ctx)

	cl := basicCluster("localhost:8888", []string{"localhost:8888"})
	host := cluster.NewSimpleHost(cl.Hosts[0], cluster.NewCluster(cl).Snapshot().ClusterInfo())

	p := connpool{
		protocol: protocol.Xprotocol,
		tlsHash:  &types.HashValue{},
	}
	p.host.Store(host)

	pMultiplex := NewPoolMultiplex(&p)
	pInst := pMultiplex.(*poolMultiplex)

	assert.Equal(t, len(pInst.activeClients), testClientNum)
	// set status for each client
	for i := 0; i < len(pInst.activeClients); i++ {
		pInst.activeClients[i].Store(types.ProtocolName("dubbo"), &activeClientMultiplex{
			state: Connected,
		})
	}

	////// scene 1, client id not previously set
	assert.True(t, pInst.CheckAndInit(ctx))
	idSetByPool := getClientIDFromDownStreamCtx(ctx)
	assert.Equal(t, int(idSetByPool), 1)

	// the id is already set, should always use the same client id
	assert.True(t, pInst.CheckAndInit(ctx))
	idSetByPool = getClientIDFromDownStreamCtx(ctx)
	assert.Equal(t, int(idSetByPool), 1)

	////// scene 2, the new request without client id
	// should use the next client
	assert.True(t, pInst.CheckAndInit(ctxNew))
	idSetByPool = getClientIDFromDownStreamCtx(ctxNew)
	assert.Equal(t, int(idSetByPool), 2)
}

func TestMultiplexParallelShutdown(t *testing.T) {
	var addr = "127.0.0.1:10086"
	go server.start(t, addr)
	defer server.stop(t)
	// wait for server to start
	time.Sleep(time.Second * 2)

	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyConfigUpStreamProtocol, string(protocol.Xprotocol))
	ctx = mosnctx.WithValue(ctx, types.ContextSubProtocol, "dubbo")

	cl := basicCluster(addr, []string{addr})
	connNum := uint32(1)
	cl.CirBreThresholds.Thresholds[0].MaxConnections = connNum

	host := cluster.NewSimpleHost(cl.Hosts[0], cluster.NewCluster(cl).Snapshot().ClusterInfo())
	p := connpool{
		protocol: protocol.Xprotocol,
		tlsHash:  &types.HashValue{},
	}
	p.host.Store(host)

	pMultiplex := NewPoolMultiplex(&p)
	pInst := pMultiplex.(*poolMultiplex)

	// init the connection
	pInst.CheckAndInit(ctx)
	// sleep to wait for the connection to be established
	time.Sleep(time.Second * 2)

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

	assert.Equal(t, len(pInst.activeClients), int(connNum))

	// parallel shutdown
	var wg = sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			pInst.Shutdown()
		}()
	}
	wg.Wait() // should not stuck here
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
		Name:        name,
		ClusterType: v2.SIMPLE_CLUSTER,
		LbType:      v2.LB_ROUNDROBIN,
		Hosts:       vhosts,
		CirBreThresholds: v2.CircuitBreakers{
			Thresholds: []v2.Thresholds{
				{MaxConnections: 10}, // this config should be read by the pool
			},
		},
	}
}
