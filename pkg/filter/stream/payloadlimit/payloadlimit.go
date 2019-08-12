package payloadlimit

import (
	"context"

	"encoding/json"

	"sofastack.io/sofa-mosn/pkg/api/v2"
	"sofastack.io/sofa-mosn/pkg/config"
	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/types"
)

type payloadLimitConfig struct {
	maxEntitySize int32
	status        int32
}

// streamPayloadLimitFilter is an implement of types.StreamReceiverFilter
type streamPayloadLimitFilter struct {
	ctx     context.Context
	handler types.StreamReceiverFilterHandler
	config  *payloadLimitConfig
	headers types.HeaderMap
}

func NewFilter(ctx context.Context, cfg *v2.StreamPayloadLimit) types.StreamReceiverFilter {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("create a new payload limit filter")
	}
	return &streamPayloadLimitFilter{
		ctx:    ctx,
		config: makePayloadLimitConfig(cfg),
	}
}

func makePayloadLimitConfig(cfg *v2.StreamPayloadLimit) *payloadLimitConfig {
	config := &payloadLimitConfig{
		maxEntitySize: cfg.MaxEntitySize,
		status:        cfg.HttpStatus,
	}
	return config
}
func parseStreamPayloadLimitConfig(c interface{}) (*payloadLimitConfig, bool) {
	conf := make(map[string]interface{})
	b, err := json.Marshal(c)
	if err != nil {
		log.DefaultLogger.Errorf("config is not a json, %v", err)
		return nil, false
	}
	json.Unmarshal(b, &conf)
	cfg, err := config.ParseStreamPayloadLimitFilter(conf)
	if err != nil {
		log.DefaultLogger.Errorf("config is not stream payload limit", err)
		return nil, false
	}
	return makePayloadLimitConfig(cfg), true
}

// ReadPerRouteConfig makes route-level configuration override filter-level configuration
func (f *streamPayloadLimitFilter) ReadPerRouteConfig(cfg map[string]interface{}) {
	if cfg == nil {
		return
	}
	if payloadLimit, ok := cfg[v2.PayloadLimit]; ok {
		if config, ok := parseStreamPayloadLimitConfig(payloadLimit); ok {
			log.DefaultLogger.Errorf("use router config to replace stream filter config, config: %v", payloadLimit)
			f.config = config
		}
	}
}

func (f *streamPayloadLimitFilter) SetReceiveFilterHandler(handler types.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *streamPayloadLimitFilter) OnReceive(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) types.StreamFilterStatus {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("payload limit stream do receive headers")
	}
	if route := f.handler.Route(); route != nil {
		// TODO: makes ReadPerRouteConfig as the StreamReceiverFilter's function
		f.ReadPerRouteConfig(route.RouteRule().PerFilterConfig())
	}
	f.headers = headers

	// buf is nil means request method is GET?
	if buf != nil && f.config.maxEntitySize != 0 {
		if buf.Len() > int(f.config.maxEntitySize) {
			if log.Proxy.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("payload size too large,data size = %d ,limit = %d",
					buf.Len(), f.config.maxEntitySize)
			}
			f.handler.RequestInfo().SetResponseFlag(types.ReqEntityTooLarge)
			f.handler.SendHijackReply(int(f.config.status), f.headers)
			return types.StreamFilterStop
		}
	}

	return types.StreamFilterContinue
}

func (f *streamPayloadLimitFilter) OnDestroy() {}
