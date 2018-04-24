package stream

import (
	"context"

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type CodecClient interface {
	types.ConnectionEventListener
	types.ReadFilter

	Id() uint64

	AddConnectionCallbacks(cb types.ConnectionEventListener)

	ActiveRequestsNum() int

	NewStream(streamId string, respDecoder types.StreamDecoder) types.StreamEncoder

	SetConnectionStats(stats *types.ConnectionStats)

	SetCodecClientCallbacks(cb CodecClientCallbacks)

	SetCodecConnectionCallbacks(cb types.StreamConnectionEventListener)

	Close()

	RemoteClose() bool
}

type CodecClientCallbacks interface {
	OnStreamDestroy()

	OnStreamReset(reason types.StreamResetReason)
}

type ProtocolStreamFactory interface {
	CreateClientStream(context context.Context, connection types.ClientConnection,
		streamConnCallbacks types.StreamConnectionEventListener,
		callbacks types.ConnectionEventListener) types.ClientStreamConnection

	CreateServerStream(context context.Context, connection types.Connection,
		callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection

	CreateBiDirectStream(context context.Context, connection types.ClientConnection,
		clientCallbacks types.StreamConnectionEventListener,
		serverCallbacks types.ServerStreamConnectionEventListener) types.ClientStreamConnection
}
