package sofarpc

import (
	"reflect"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"time"
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
	healthCheckReq bool
	// callbacks
	cb types.StreamDecoderFilterCallbacks
}

func NewHealthCheckFilter(config v2.HealthCheckFilter) *healthCheckFilter {
	return &healthCheckFilter{
		passThrough:                  config.PassThrough,
		cacheTime:                    config.CacheTime,
		clusterMinHealthyPercentages: config.ClusterMinHealthyPercentage,
	}
}

func (f *healthCheckFilter) DecodeHeaders(headers map[string]string, endStream bool) types.FilterHeadersStatus {
	if cmdCodeStr, ok := headers[sofarpc.SofaPropertyHeader("cmdcode")]; ok {
		cmdCode := sofarpc.ConvertPropertyValue(cmdCodeStr, reflect.Int16)

		if cmdCode == 0 {
			protocolStr := headers[sofarpc.SofaPropertyHeader("protocol")]
			f.protocol = sofarpc.ConvertPropertyValue(protocolStr, reflect.Uint8).(byte)
			f.healthCheckReq = true
			f.cb.RequestInfo().SetHealthCheck(true)

			f.intercept = true
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

	resp := &sofarpc.BoltResponseCommand{
		Protocol: f.protocol,
		CmdType:  byte(2),
		CmdCode:  int16(0),
		// todo: fill in more properties...
	}

	f.cb.EncodeHeaders(resp, true)
}

func (f *healthCheckFilter) SetDecoderFilterCallbacks(cb types.StreamDecoderFilterCallbacks) {
	f.cb = cb
}

func (f *healthCheckFilter) OnDestroy() {}

// ~~ factory
type HealthCheckFilterConfigFactory struct {
	FilterConfig v2.HealthCheckFilter
}

func (f *HealthCheckFilterConfigFactory) CreateFilterChain(callbacks types.FilterChainFactoryCallbacks) {
	filter := NewHealthCheckFilter(f.FilterConfig)
	callbacks.AddStreamDecoderFilter(filter)
}
