package types

// common flags for transfer
const (
	GracefulRestart = "_MOSN_GRACEFUL_RESTART"
	InheritFd       = "_MOSN_INHERIT_FD"
)

// Stoppable is an interface for server/listener, which can be stopped when transfer
type Stoppable interface {
	Close() error
}
