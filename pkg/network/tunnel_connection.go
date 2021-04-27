package network

import (
	"errors"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

var _ types.ClientConnection = (*TunnelAgentConnection)(nil)

type TunnelAgentConnection struct {
	api.Connection
}

func (cc *TunnelAgentConnection) Connect() (err error) {
	if cc.State() == api.ConnClosed {
		return errors.New("")
	}
	return nil
}

func CreateTunnelAgentConnection(conn api.Connection) *TunnelAgentConnection {
	return &TunnelAgentConnection{
		Connection: conn,
	}
}
