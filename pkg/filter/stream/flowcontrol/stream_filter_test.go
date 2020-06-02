package flowcontrol

import (
	"context"
	"testing"

	"mosn.io/mosn/pkg/variable"

	"mosn.io/api"

	mosnctx "mosn.io/mosn/pkg/context"

	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/types"
)

const (
	HTTP1 types.ProtocolName = "Http1"
	Dubbo types.ProtocolName = "Dubbo"
)

func TestStreamFilter(t *testing.T) {
	mockConfig := &Config{
		GlobalSwitch: false,
		Monitor:      false,
		KeyType:      "PATH",
		Rules: []*flow.FlowRule{
			&flow.FlowRule{
				ID:              0,
				Resource:        "/http",
				MetricType:      1,
				Count:           1,
				ControlBehavior: 0,
			},
		},
	}
	// global switch disabled
	sf := NewStreamFilter(&DefaultCallbacks{config: mockConfig})
	status := sf.OnReceive(context.Background(), nil, nil, nil)
	assert.Equal(t, api.StreamFilterContinue, status)

	// global switch enabled
	mockConfig.GlobalSwitch = true
	sf = NewStreamFilter(&DefaultCallbacks{config: mockConfig})
	status = sf.OnReceive(context.Background(), nil, nil, nil)
	assert.Equal(t, api.StreamFilterContinue, status)

	ctx := context.Background()
	ctx = mosnctx.WithValue(ctx, types.ContextKeyDownStreamProtocol, HTTP1)

	m := make(map[string]string)
	m["http_request_path"] = "/http"
	m["dubbo_request_path"] = "/dubbo"
	for k := range m {
		// register test variable
		variable.RegisterVariable(variable.NewBasicVariable(k, nil,
			func(ctx context.Context, variableValue *variable.IndexedValue, data interface{}) (s string, err error) {
				return m[k], nil
			}, nil, 0))
	}
	variable.RegisterProtocolResource(HTTP1, api.PATH, "http_request_path")
	status = sf.OnReceive(ctx, nil, nil, nil)
	assert.NotEmpty(t, status)
}
