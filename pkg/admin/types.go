package admin

import (
	. "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
)

type Config interface {
	GetAdmin() *Admin
}
