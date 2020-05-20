package handler

import (
	"context"
	"fmt"

	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/types"
)

func init() {
	XHandler[dubbo.ProtocolName] = GetDubboRouteHandler
	CustomerPort = append(CustomerPort, dubboPort, dubboMosnConsumerPort, dubboMosnProviderPort)
}

const (
	dubboClusterPre = "outbound|%d|%s|dubbo-%s-%s"

	// consumer  -> 127.0.0.1:20881(mosn-consumer)->provider:20882(mosn-provider)->127.0.0.1:20880
	dubboPort             = 20880
	dubboMosnConsumerPort = 20881
	dubboMosnProviderPort = 20882
)

type dubboHandler struct {
	route  api.Route
	header api.HeaderMap
}

func GetDubboRouteHandler(route api.Route, header api.HeaderMap) types.RouteHandler {
	return &dubboHandler{route: route, header: header}
}

func (d *dubboHandler) IsAvailable(ctx context.Context, manager types.ClusterManager) (types.ClusterSnapshot, types.HandlerStatus) {
	if d.route == nil {
		return nil, types.HandlerNotAvailable
	}

	service, ok := d.header.Get(dubbo.ServiceNameHeader)
	if !ok {
		return nil, types.HandlerNotAvailable
	}
	method, ok := d.header.Get(dubbo.MethodNameHeader)
	if !ok {
		return nil, types.HandlerNotAvailable
	}
	// TODO: support through subset build cluster
	var subset string

	listenerPort, ok := ctx.Value(types.ContextKeyListenerPort).(int)
	if !ok {
		return nil, types.HandlerNotAvailable
	}

	var clusterName string
	if listenerPort == dubboMosnProviderPort {
		clusterName = d.Route().RouteRule().ClusterName()
	} else {
		clusterName = fmt.Sprintf(dubboClusterPre, dubboMosnProviderPort, subset, service, method)
	}
	// clusterName = d.Route().RouteRule().ClusterName()

	snapshot := manager.GetClusterSnapshot(context.Background(), clusterName)
	if snapshot == nil {
		return nil, types.HandlerNotAvailable
	}
	return snapshot, types.HandlerAvailable
}

func (d *dubboHandler) Route() api.Route {
	return d.route
}
