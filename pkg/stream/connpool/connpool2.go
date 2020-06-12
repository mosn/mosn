package connpool

import (
	"context"
	"errors"
	"math"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/pkg/buffer"
	"sync/atomic"
)

// NewConn returns a simplified connpool
func NewConn(hostAddr string, reconnectTryTimes int, heartBeatCreator func() KeepAlive2, readFilters []api.ReadFilter, autoReconnectWhenClose bool) types.Connection {
	// use host addr as cluster name, for the count of metrics
	cl := basicCluster(hostAddr, []string{hostAddr})
	host := cluster.NewSimpleHost(cl.Hosts[0], cluster.NewCluster(cl).Snapshot().ClusterInfo())

	// if user configure this to -1, then retry is unlimited
	if reconnectTryTimes == -1 {
		reconnectTryTimes = math.MaxInt32
	}

	p := &connpool{
		supportTLS:  host.SupportTLS(),
		protocol:    "",
		idleClients: make(map[api.Protocol][]*activeClient),

		useDefaultCodec:        false,
		heartBeatCreator:       heartBeatCreator,
		forceMultiplex:         true,
		autoReconnectWhenClose: autoReconnectWhenClose,
		reconnTryTimes:         reconnectTryTimes,
		readFilters:            readFilters,
	}

	p.host.Store(host)
	p.newActiveClient(context.TODO(),"")

	return p
}

// write to client
func (p *connpool) Write(buf ...buffer.IoBuffer) error {
	c, reason := p.GetActiveClient(context.Background(), "")
	if reason != "" {
		return errors.New("get client failed" + string(reason))
	}

	cli := c.(*activeClient)

	err := cli.host.Connection.Write(buf...)
	if err != nil {
		cli.Reconnect()
	}

	return err
}

// Available current available to send request
// WARNING, this api is only for msg
// WARNING, dont use it any other scene
func (p * connpool) Available() bool {
	var subProto api.Protocol = ""

	// if pool was destroyed
	if atomic.LoadUint64(&p.destroyed) == 1 {
		return false
	}

	// if there is no clients
	if len(p.idleClients) == 0 {
		return false
	}

	// if there is no client in such protocol
	lastIdx := len(p.idleClients[subProto]) - 1
	if lastIdx < 0 {
		return false
	}

	// if the client was closed
	if p.idleClients[subProto][lastIdx].closed {
		return false
	}

	// if the client has gone away
	if atomic.LoadUint32(&p.idleClients[subProto][lastIdx].goaway) == 1 {
		return false
	}

	// if we are reconnecting
	if atomic.LoadUint64(&p.idleClients[subProto][lastIdx].reconnectState) == connecting {
		return false
	}

	return true
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
		Name:                 name,
		ClusterType:          v2.SIMPLE_CLUSTER,
		LbType:               v2.LB_ROUNDROBIN,
		ConnBufferLimitBytes: 16 * 1026,
		Hosts:                vhosts,
	}
}
