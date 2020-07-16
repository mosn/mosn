package mirror

import (
	"context"

	"mosn.io/api"
)

func init() {
	api.RegisterStream("mirror", NewMirrorConfig)
}

func NewMirrorConfig(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	return &config{}, nil
}

type config struct{}

func (c *config) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	m := &mirror{}
	callbacks.AddStreamReceiverFilter(m, api.AfterRoute)
}
