package handler

import (
	"context"
	"strings"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

type handler func(route api.Route, header api.HeaderMap) types.RouteHandler

var (
	XHandler = make(map[string]handler)

	// CustomerPort use xDS shuold filter this 0.0.0.0_port
	CustomerPort = make([]int, 0, 2)
)

func GetRouteHandler(ctx context.Context, route api.Route, header api.HeaderMap) types.RouteHandler {
	protocal, ok := ctx.Value(types.ContextSubProtocol).(string)
	if ok {
		if h, ok := XHandler[strings.ToLower(protocal)]; ok {
			return h(route, header)
		}
	}

	return &simpleHandler{route: route}
}
