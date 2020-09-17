package types

// PoolMode is whether PingPong or multiplex
type PoolMode int

const (
	PingPong PoolMode = iota
	Multiplex
	TCP
)
