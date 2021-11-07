package proxy

import (
	"context"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/types"
	"testing"
)

func TestWebsocketProxy(t *testing.T) {
	proxy := &proxy{
		config:              &v2.Proxy{},
		routersWrapper:      nil,
		clusterManager:      &mockClusterManager{},
		readCallbacks:       &mockReadFilterCallbacks{},
		stats:               globalStats,
		listenerStats:       newListenerStats("test"),
		serverStreamConn:    &mockServerConn{},
		routeHandlerFactory: router.DefaultMakeHandler,
	}
	s := newActiveStream(context.Background(), proxy, nil, nil)

	args := &types.ProxyWebsocketArgs{}
	wp := createWebsocketProxy(s, args)
	wp.handleWebsocket()

}
