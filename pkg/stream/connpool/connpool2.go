package connpool

import (
	"context"
	"errors"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/pkg/buffer"
)

// NewConn returns a simplified connpool
func NewConn(hostAddr string, reconnectTryTimes int, heartBeatCreator func() KeepAlive2, readFilters []api.ReadFilter) types.Connection {
	// use host addr as cluster name, for the count of metrics
	cl := basicCluster(hostAddr, []string{hostAddr})
	host := cluster.NewSimpleHost(cl.Hosts[0], cluster.NewCluster(cl).Snapshot().ClusterInfo())

	p := &connpool{
		supportTLS:  host.SupportTLS(),
		protocol:    "",
		idleClients: make(map[api.Protocol][]*activeClient),

		useDefaultCodec:        false,
		heartBeatCreator:       heartBeatCreator,
		forceMultiplex:         true,
		autoReconnectWhenClose: true,
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
