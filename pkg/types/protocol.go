package types

type Protocol string

type PoolFailureReason string

const (
	Overflow          PoolFailureReason = "Overflow"
	ConnectionFailure PoolFailureReason = "ConnectionFailure"
)

type ConnectionPool interface {
	Protocol() Protocol

	DrainConnections()

	NewStream(streamId string, responseDecoder StreamDecoder, cb PoolEventListener) Cancellable

	Close()
}

type PoolEventListener interface {
	OnPoolFailure(streamId string, reason PoolFailureReason, host Host)

	OnPoolReady(streamId string, requestEncoder StreamEncoder, host Host)
}

type Cancellable interface {
	Cancel()
}
