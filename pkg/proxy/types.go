package proxy

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type CodecClient interface {
	types.ConnectionCallbacks
	types.ReadFilter

	Id() uint64

	AddConnectionCallbacks(cb types.ConnectionCallbacks)

	ActiveRequestsLen() int

	NewStream(streamId uint32, respDecoder types.StreamDecoder) types.StreamEncoder

	SetConnectionStats(stats types.ConnectionStats)

	SetCodecClientCallbacks(cb CodecClientCallbacks)

	SetCodecConnectionCallbacks(cb types.StreamConnectionCallbacks)

	Close()

	RemoteClose() bool
}

type CodecClientCallbacks interface {
	OnStreamDestroy()

	OnStreamReset(reason types.StreamResetReason)
}