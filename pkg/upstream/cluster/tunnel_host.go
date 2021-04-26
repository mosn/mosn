package cluster

import (
	"context"
	"net"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

var _ types.Host = (*TunnelHost)(nil)

type TunnelHost struct {
	simpleHost
	conn api.Connection
}

var _ types.ClientConnection = (*TunnelAgentConnection)(nil)

type TunnelAgentConnection struct {

}

func (t TunnelHost) Hostname() string {
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
}

func (t TunnelHost) Address() net.Addr {
	return t.conn.RemoteAddr()
}

func (t TunnelHost) Config() v2.Host {
	panic("implement me")
}
