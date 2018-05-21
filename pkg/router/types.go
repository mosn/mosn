package router

import "gitlab.alipay-inc.com/afe/mosn/pkg/types"

type Matchable interface {
	Match(headers map[string]string) (types.Route)
	
}

type RouterInfo interface{
	GetRouterName()string
}

type RouteBase interface {
	types.Route
	types.RouteRule
	Matchable
	RouterInfo
}