package websocket

import (
	"strings"

	"github.com/valyala/fasthttp"
)

const (
	protocolWebSocket= "websocket"
)

// IsUpgradeWebSocket whether client request for websocket protocol
func IsUpgradeWebSocket(req *fasthttp.Request) bool {
	if !req.Header.IsGet() {
		return false
	}
	if strings.ToLower(string(req.Header.Peek("Upgrade"))) != protocolWebSocket {
		return false
	}
	if !strings.Contains(strings.ToLower(string(req.Header.Peek("Connection"))), "upgrade") {
		return false
	}
	return true
}