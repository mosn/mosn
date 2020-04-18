package handler

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

type simpleHandler struct {
	route api.Route
}

func (h *simpleHandler) IsAvailable(ctx context.Context, manager types.ClusterManager) (types.ClusterSnapshot, types.HandlerStatus) {
	if h.route == nil {
		return nil, types.HandlerNotAvailable
	}
	clusterName := h.Route().RouteRule().ClusterName()
	snapshot := manager.GetClusterSnapshot(context.Background(), clusterName)
	return snapshot, types.HandlerAvailable
}

func (h *simpleHandler) Route() api.Route {
	return h.route
}
