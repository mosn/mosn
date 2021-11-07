package types

import (
	"bufio"

	"github.com/valyala/fasthttp"
)

type ProxyArgsKeyType string
type ProxyType string

const (
	ProxyWebsocketArgKey ProxyArgsKeyType = "websocket_proxy_args"
	ProxyTypeWebsocket   ProxyType        = "websocket"
)

type WebsocketProxyArgs interface {
	GetType() ProxyType
}

type ProxyWebsocketArgs struct {
	CurrentBufferReader *bufio.Reader
	Request             *fasthttp.Request
}

func (wp *ProxyWebsocketArgs) GetType() ProxyType {
	return ProxyTypeWebsocket
}
