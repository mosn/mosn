package mirror

import (
	"context"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

var (
	defaultAmplification = 1
	amplificationKey     = "amplification"
)

func init() {
	api.RegisterStream(v2.Mirror, NewMirrorConfig)
}

func NewMirrorConfig(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	c := &config{
		Amplification: defaultAmplification,
	}

	if ampValue, ok := conf[amplificationKey]; ok {
		if amp, ok := ampValue.(float64); ok && amp > 0 {
			c.Amplification = int(amp)
		}
	}
	return c, nil
}

type config struct {
	Amplification int `json:"amplification,omitempty"`
}

func (c *config) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	m := &mirror{amplification: c.Amplification}
	callbacks.AddStreamReceiverFilter(m, api.AfterRoute)
}
