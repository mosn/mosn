package main

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/types"
)

type extendHandler struct {
	headers api.HeaderMap
	routers types.Routers
	route   api.Route
}

func (h *extendHandler) IsAvailable(ctx context.Context, manager types.ClusterManager) (types.ClusterSnapshot, types.HandlerStatus) {
	rs := h.routers.MatchAllRoutes(ctx, h.headers)
	for _, r := range rs {
		name := r.RouteRule().ClusterName(ctx)
		snap := manager.GetClusterSnapshot(ctx, name)
		// Verify is cluster config exists hosts
		if snap.IsExistsHosts(nil) {
			h.route = r
			return snap, types.HandlerAvailable
		}
	}
	return nil, types.HandlerNotAvailable
}

func (h *extendHandler) Route() api.Route {
	return h.route
}

func MakeExtendHandler(ctx context.Context, headers api.HeaderMap, routers types.Routers) types.RouteHandler {
	return &extendHandler{
		headers: headers,
		routers: routers,
	}
}

func init() {
	router.RegisterMakeHandler("check-handler", MakeExtendHandler, false)
}
