package types

type Protocol string

type PoolFailureReason string

const (
	Overflow          PoolFailureReason = "Overflow"
	ConnectionFailure PoolFailureReason = "ConnectionFailure"
)

type ConnectionPool interface {
	Protocol() Protocol

	AddDrainedCallback(cb func())

	DrainConnections()

	NewStream(streamId uint32, responseDecoder StreamDecoder, cb PoolCallbacks) Cancellable

	Close()
}

type PoolCallbacks interface {
	OnPoolFailure(streamId uint32, reason PoolFailureReason, host Host)

	OnPoolReady(streamId uint32, encoder StreamEncoder, host Host)
}

type Cancellable interface {
	Cancel()
}
