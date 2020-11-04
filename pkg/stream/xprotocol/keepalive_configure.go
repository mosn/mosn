package xprotocol

import (
	"sync/atomic"
)

// KeepaliveConfig is the the config for xprotocol keepalive
// the config parameter can be set from the outside system
type KeepaliveConfig struct {
	// the next hb will be sent after tick_count_if_fail * conn_read_timeout if the current hb fails
	TickCountIfFail uint32
	// the next hb will be sent after tick_count_if_succ * conn_read_timeout if the current hb succs
	TickCountIfSucc uint32
	// if hb fails in a line, and count = fail_count_to_close, close this connection
	FailCountToClose uint32
}

var xprotoKeepaliveConfig atomic.Value

// DefaultKeepaliveConfig keeps the save with previous behavior
var DefaultKeepaliveConfig = KeepaliveConfig{
	TickCountIfFail:  1,
	TickCountIfSucc:  1,
	FailCountToClose: 6,
}

func init() {
	RefreshKeepaliveConfig(DefaultKeepaliveConfig)
}

// RefreshKeepaliveConfig refresh the keepalive config
func RefreshKeepaliveConfig(c KeepaliveConfig) {
	xprotoKeepaliveConfig.Store(c)
}
