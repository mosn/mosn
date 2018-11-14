package faultinject

import (
	"github.com/alipay/sofa-mosn/pkg/types"
)

// this file mocks the interface that used for test
// only implement the function that used in test
type mockStreamReceiverFilterCallbacks struct {
	types.StreamReceiverFilterCallbacks
	route      *mockRoute
	hijackCode int
	info       *mockRequestInfo
	called     chan int
}

func (cb *mockStreamReceiverFilterCallbacks) Route() types.Route {
	return cb.route
}
func (cb *mockStreamReceiverFilterCallbacks) RequestInfo() types.RequestInfo {
	return cb.info
}
func (cb *mockStreamReceiverFilterCallbacks) ContinueDecoding() {
	cb.called <- 1
}
func (cb *mockStreamReceiverFilterCallbacks) SendHijackReply(code int, headers types.HeaderMap) {
	cb.hijackCode = code
	cb.called <- 1
}

type mockRoute struct {
	types.Route
	rule *mockRouteRule
}

func (r *mockRoute) RouteRule() types.RouteRule {
	return r.rule
}

type mockRouteRule struct {
	types.RouteRule
	clustername string
	config      map[string]interface{}
}

func (r *mockRouteRule) ClusterName() string {
	return r.clustername
}
func (r *mockRouteRule) PerFilterConfig() map[string]interface{} {
	return r.config
}

type mockRequestInfo struct {
	types.RequestInfo
	flag types.ResponseFlag
}

func (info *mockRequestInfo) SetResponseFlag(flag types.ResponseFlag) {
	info.flag = flag
}
