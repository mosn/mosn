package sofarpc

import (
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol/rpc/sofarpc"
	"mosn.io/pkg/buffer"
)

type SendFilter struct {
	config  *v2.FaultToleranceFilterConfig
	handler api.StreamSenderFilterHandler
}

func NewSendFilter(config *v2.FaultToleranceFilterConfig) *SendFilter {
	filter := &SendFilter{
		config: config,
	}
	return filter
}

func (f *SendFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	if fault_tolerance_strategy.GetStrategy() == constant.FAULT_TOLERANCE_STRATEGY_OFF {
		return api.StreamFilterContinue
	}

	traceAndRpcId := ""
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		traceAndRpcId = router_common.GetTraceAndRpcId(headers)
	}

	if _, ok := headers.(*sofarpc.BoltResponse); !ok {
		if _, ok := headers.(*sofarpc.BoltRequest); !ok {
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[%s][Tolerance][FaultToleranceSendFilter] headers is not instance of bolt.", traceAndRpcId)
			}
			return api.StreamFilterContinue
		}
	}

	if ok, host, app, zone, address, status := GetInvokeInfo(f.handler.RequestInfo()); ok {
		stat := f.invocationFactory.GetInvocationStat(host, app, zone, address)
		config := fault_tolerance_rule.GetFaultToleranceStoreInstance().GetRule(app)
		stat.Call(config.IsException(status))

	}

	return api.StreamFilterContinue
}

func (f *SendFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.handler = handler
}

func (f *SendFilter) OnDestroy() {

}
