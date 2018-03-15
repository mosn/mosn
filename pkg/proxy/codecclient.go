package proxy

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"container/list"
)

// CodecClient
type BaseCodeClient struct {
	Protocol   types.Protocol
	Connection types.ClientConnection
	Host       types.HostInfo
	Codec      types.ClientStreamConnection

	ActiveRequests            *list.List
	CodecCallbacks            types.StreamConnectionCallbacks
	CodecClientCallbacks      CodecClientCallbacks
	StreamConnectionCallbacks types.StreamConnectionCallbacks
	ConnectedFlag             bool
	RemoteCloseFlag           bool
}

func (c *BaseCodeClient) Id() uint64 {
	return c.Connection.Id()
}

func (c *BaseCodeClient) AddConnectionCallbacks(cb types.ConnectionCallbacks) {
	c.Connection.AddConnectionCallbacks(cb)
}

func (c *BaseCodeClient) ActiveRequestsLen() int {
	return c.ActiveRequests.Len()
}

func (c *BaseCodeClient) SetConnectionStats(stats types.ConnectionStats) {
	// todo
}

func (c *BaseCodeClient) SetCodecClientCallbacks(cb CodecClientCallbacks) {
	c.CodecClientCallbacks = cb
}

func (c *BaseCodeClient) SetCodecConnectionCallbacks(cb types.StreamConnectionCallbacks) {
	c.StreamConnectionCallbacks = cb
}

func (c *BaseCodeClient) RemoteClose() bool {
	return c.RemoteCloseFlag
}
