package mirror

import (
	"context"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

var (
	defaultAmplification = 1
	amplificationKey     = "amplification"
	broadcastKey         = "broadcast"
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
	if broadcast, ok := conf[broadcastKey]; ok {
		c.BroadCast = broadcast.(bool)
	}
	return c, nil
}

type config struct {
	Amplification int  `json:"amplification,omitempty"`
	BroadCast     bool `json:"broadcast,omitempty"`
}

func (c *config) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	m := &mirror{
		amplification: c.Amplification,
		broadcast:     c.BroadCast,
	}
	callbacks.AddStreamReceiverFilter(m, api.AfterRoute)
}
