package types

import "time"

type KeepAlive interface {
	Start()
	SendKeepAlive()
	GetTimeout() time.Duration
	HandleTimeout(id uint64)
	HandleSuccess(id uint64)
	AddCallback(cb KeepAliveCallback)
}

type KeepAliveStatus int

const (
	KeepAliveSuccess KeepAliveStatus = iota
	KeepAliveTimeout
)

// KeepAliveCallback is a callback when keep alive handle response/timeout
type KeepAliveCallback func(KeepAliveStatus)
