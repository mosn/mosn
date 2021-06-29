package jaeger

import (
	"fmt"

	"mosn.io/api"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
)

const (
	DriverName          = "jaeger"
	HeaderRouteMatchKey = "service"
)

func init() {
	trace.RegisterDriver(DriverName, NewJaegerImpl())
}

type holder struct {
	api.Tracer
	api.TracerBuilder
}

type jaegerDriver struct {
	tracers map[types.ProtocolName]*holder
}

func (d *jaegerDriver) Init(traceCfg map[string]interface{}) error {
	for proto, holder := range d.tracers {
		tracer, err := holder.TracerBuilder(traceCfg)
		if err != nil {
			return fmt.Errorf("build tracer for %v error, %s", proto, err)
		}

		holder.Tracer = tracer
	}

	return nil
}

func (d *jaegerDriver) Register(proto types.ProtocolName, builder api.TracerBuilder) {
	d.tracers[proto] = &holder{
		TracerBuilder: builder,
	}
}

func (d *jaegerDriver) Get(proto types.ProtocolName) api.Tracer {
	if holder, ok := d.tracers[proto]; ok {
		return holder.Tracer
	}
	return nil
}

// NewJaegerImpl create jaeger driver
func NewJaegerImpl() api.Driver {
	return &jaegerDriver{
		tracers: make(map[types.ProtocolName]*holder),
	}
}
