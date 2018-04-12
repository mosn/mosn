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

	NewStream(streamId string, responseDecoder StreamDecoder, cb PoolCallbacks) Cancellable

	Close()
}

type PoolCallbacks interface {
	OnPoolFailure(streamId string, reason PoolFailureReason, host Host)

	OnPoolReady(streamId string, requestEncoder StreamEncoder, host Host)
}

type Cancellable interface {
	Cancel()
}
