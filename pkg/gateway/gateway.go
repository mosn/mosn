package gateway

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/filter"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	filter.RegisterStream("gateway", CreateGatewayFactory)
}

type FilterConfigFactory struct{}

func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks types.StreamFilterChainFactoryCallbacks) {
	filter := NewGetway()
	callbacks.AddStreamReceiverFilter(filter)
}

func CreateGatewayFactory(conf map[string]interface{}) (types.StreamFilterChainFactory, error) {
	return &FilterConfigFactory{}, nil
}

func NewGetway() types.StreamReceiverFilter {
	return &gateway{}
}

type gateway struct {
	handler types.StreamReceiverFilterHandler
}

func (g *gateway) SetReceiveFilterHandler(handler types.StreamReceiverFilterHandler) {
	g.handler = handler
}

// 如果只有Header, 执行完handle以后 需要返回Continue
// 如果有Data，Header需要返回Stop,中止后续的流程
func (g *gateway) OnReceiveHeaders(headers types.HeaderMap, endStream bool) types.StreamHeadersFilterStatus {
	if endStream {
		g.handle()
		if true { //需要重新走路由, 可以考虑做成handle的返回值
			g.handler.ContinueReceiving()
			return types.StreamHeadersFilterStop
		}
		return types.StreamHeadersFilterContinue
	}
	return types.StreamHeadersFilterStop
}

func (g *gateway) OnReceiveData(buf types.IoBuffer, endStream bool) types.StreamDataFilterStatus {
	if endStream {
		g.handle()
		return types.StreamDataFilterContinue
	}
	// 保留Data, 等待Trailer
	// 不能返回Stop，返回Stop以后，Data会被Reset
	return types.StreamDataFilterStopAndBuffer
}

func (g *gateway) OnReceiveTrailers(trailers types.HeaderMap) types.StreamTrailersFilterStatus {
	// 网关的Trailer 一定只能返回Continue
	g.handle()
	return types.StreamTrailersFilterContinue
}

func (g *gateway) OnDestroy() {}

func (g *gateway) handle() {
	// 网关的Handler行为
	headers := g.handler.GetRequestHeader()
	//data := g.handler.GetRequestData()
	//if data != nil {
	// 处理
	//}
	// 处理
	headers.Set("Test-Gateway", "Test")
	g.handler.SetRequestHeader(headers)
	// g.handler.SetRequestData(data)

}
