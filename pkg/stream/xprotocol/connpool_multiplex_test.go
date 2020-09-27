package xprotocol

import (
	"context"
	"github.com/stretchr/testify/assert"
	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"testing"
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
