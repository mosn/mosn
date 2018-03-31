package stream

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/stream/faultinject"
)

type FaultInjectFilterConfigFactory struct {
	FaultInject *v2.FaultInject
}

func (f *FaultInjectFilterConfigFactory) CreateFilterChain(callbacks types.FilterChainFactoryCallbacks) {
	filter := faultinject.NewFaultInjectFilter(f.FaultInject)
	callbacks.AddStreamDecoderFilter(filter)
}
