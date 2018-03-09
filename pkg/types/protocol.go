package types

type Protocol string

type PoolFailureReason string

const (
	Overflow PoolFailureReason = "Overflow"
	ConnectionFailure PoolFailureReason = "ConnectionFailure"
)

type ConnectionPool interface {
	Protocol() string

	AddDrainedCallback(cb func())

	DrainConnections()

	NewStream(responseDecoder StreamDecoder, cb PoolCallbacks) Cancellable
}

type PoolCallbacks interface {
	OnPoolFailure(reason string, host Host)

	onPoolReady(encoder StreamEncoder, host Host)
}

type Cancellable interface {
	Cancel()
}


