package router

import "gitlab.alipay-inc.com/afe/mosn/pkg/types"

type Matchable interface {
	Match(headers map[string]string,randomValue uint64) (types.Route,string)
	
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