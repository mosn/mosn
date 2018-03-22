package proxy

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func parseProxyTimeout(route types.Route, headers map[string]string) *ProxyTimeout {
	timeout := &ProxyTimeout{}
	timeout.GlobalTimeout = route.RouteRule().GlobalTimeout()
	timeout.TryTimeout = route.RouteRule().Policy().RetryPolicy().TryTimeout()

	// todo: check global timeout in request headers
	// todo: check per try timeout in request headers

	if timeout.TryTimeout >= timeout.GlobalTimeout {
		timeout.TryTimeout = 0
	}

	return timeout
}
