package sofarpc

import (
	"reflect"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"time"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
)

// todo: support cached pass through

// types.StreamEncoderFilter
type healthCheckFilter struct {
	// config
	passThrough                  bool
	cacheTime                    time.Duration
	clusterMinHealthyPercentages map[string]float32
	// request properties
	intercept      bool
	protocol       byte
	requestId      uint32
	healthCheckReq bool
	// callbacks
	cb types.StreamDecoderFilterCallbacks
}

func NewHealthCheckFilter(config *v2.HealthCheckFilter) *healthCheckFilter {
	return &healthCheckFilter{
		passThrough:                  config.PassThrough,
		cacheTime:                    config.CacheTime,
		clusterMinHealthyPercentages: config.ClusterMinHealthyPercentage,
	}
}

func (f *healthCheckFilter) DecodeHeaders(headers map[string]string, endStream bool) types.FilterHeadersStatus {
	if cmdCodeStr, ok := headers[sofarpc.SofaPropertyHeader("cmdcode")]; ok {
		cmdCode := sofarpc.ConvertPropertyValue(cmdCodeStr, reflect.Int16)

		//sofarpc.HEARTBEAT(0) is equal to sofarpc.TR_HEARTBEAT(0)
		if cmdCode == sofarpc.HEARTBEAT {
			protocolStr := headers[sofarpc.SofaPropertyHeader("protocol")]
			f.protocol = sofarpc.ConvertPropertyValue(protocolStr, reflect.Uint8).(byte)
			requestIdStr := headers[sofarpc.SofaPropertyHeader("requestid")]
			f.requestId = sofarpc.ConvertPropertyValue(requestIdStr, reflect.Uint32).(uint32)
			f.healthCheckReq = true
			f.cb.RequestInfo().SetHealthCheck(true)

			if !f.passThrough {
				f.intercept = true
			}
			endStream = true
		}
	}

	if endStream && f.intercept {
		f.handleIntercept()
	}

	if f.intercept {
		return types.FilterHeadersStatusStopIteration
	} else {
		return types.FilterHeadersStatusContinue
	}
}

func (f *healthCheckFilter) DecodeData(buf types.IoBuffer, endStream bool) types.FilterDataStatus {
	if endStream && f.intercept {
		f.handleIntercept()
	}

	if f.intercept {
		return types.FilterDataStatusStopIterationNoBuffer
	} else {
		return types.FilterDataStatusContinue
	}
}

func (f *healthCheckFilter) DecodeTrailers(trailers map[string]string) types.FilterTrailersStatus {
	if f.intercept {
		f.handleIntercept()
	}

	if f.intercept {
		return types.FilterTrailersStatusStopIteration
	} else {
		return types.FilterTrailersStatusContinue
	}
}

func (f *healthCheckFilter) handleIntercept() {
	// todo: cal status based on cluster healthy host stats and f.clusterMinHealthyPercentages

	var resp interface{}

	//TODO add protocl-level interface for heartbeat process, like Protocols.TriggerHeartbeat(protocolCode, requestId)&Protocols.ReplyHeartbeat(protocolCode, requestId)
	switch {
	//case f.protocol == sofarpc.PROTOCOL_CODE:
		//resp = codec.NewTrHeartbeatAck( f.requestId)
	case f.protocol == sofarpc.PROTOCOL_CODE_V1 || f.protocol == sofarpc.PROTOCOL_CODE_V2:
		//boltv1 and boltv2 use same heartbeat struct as BoltV1
		resp = codec.NewBoltHeartbeatAck( f.requestId)
	default:
		log.DefaultLogger.Debugf("Unknown protocol code: [", f.protocol, "] while intercept healthcheck.")
	}

	f.cb.EncodeHeaders(resp, true)
}


func (f *healthCheckFilter) SetDecoderFilterCallbacks(cb types.StreamDecoderFilterCallbacks) {
	f.cb = cb
}

func (f *healthCheckFilter) OnDestroy() {}

// ~~ factory
type HealthCheckFilterConfigFactory struct {
	FilterConfig *v2.HealthCheckFilter
}

func (f *HealthCheckFilterConfigFactory) CreateFilterChain(callbacks types.FilterChainFactoryCallbacks) {
	filter := NewHealthCheckFilter(f.FilterConfig)
	callbacks.AddStreamDecoderFilter(filter)
}
