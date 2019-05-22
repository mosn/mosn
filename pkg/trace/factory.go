package trace

import (
	"sofastack.io/sofa-mosn/pkg/types"
)

type TracerBuilder func() types.Tracer

var (
	TracerFactory = make(map[string]TracerBuilder)
)

func RegisterTracerBuilder(tracer string, builder TracerBuilder) {
	TracerFactory[tracer] = builder
}

func CreateTracer(tracer string) types.Tracer {
	if builder, ok := TracerFactory[tracer]; ok {
		return builder()
	} else {
		return nil
	}
}
