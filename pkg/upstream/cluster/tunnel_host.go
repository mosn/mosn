package cluster

import (
	"context"
	"net"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

var _ types.Host = (*TunnelHost)(nil)

type TunnelHost struct {
	types.Host
	conn types.ClientConnection
}

func (t TunnelHost) AddressString() string {
	return t.conn.RemoteAddr().String()
}

func (t TunnelHost) CreateConnection(context context.Context) types.CreateConnectionData {
	return types.CreateConnectionData{
		Connection: t.conn,
	}
}

func (t TunnelHost) CreateUDPConnection(context context.Context) types.CreateConnectionData {
	return types.CreateConnectionData{
		Connection: t.conn,
	}
}

func (t TunnelHost) Address() net.Addr {
	return t.conn.RemoteAddr()
}

func NewTunnelHost(config v2.Host, clusterInfo types.ClusterInfo, connection types.ClientConnection) *TunnelHost {
	return &TunnelHost{
		Host: NewSimpleHost(config, clusterInfo),
		conn: connection,
	}
}
