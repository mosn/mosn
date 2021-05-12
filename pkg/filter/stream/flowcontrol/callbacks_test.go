package flowcontrol

import (
	"context"
	"testing"

	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"

	"github.com/stretchr/testify/assert"
)

func TestDefaultCallbacks(t *testing.T) {
	mockConfig := &Config{
		GlobalSwitch: false,
		Monitor:      false,
		KeyType:      "PATH",
	}
	cb := &DefaultCallbacks{config: mockConfig}
	assert.False(t, cb.Enabled())
	mockConfig.GlobalSwitch = true
	assert.True(t, cb.Enabled())
}

func TestCallbacksRegistry(t *testing.T) {
	cfg := defaultConfig()
	cb := &DefaultCallbacks{config: defaultConfig()}
	RegisterCallbacks("default", cb)
	assert.NotNil(t, GetCallbacksByConfig(cfg))
	cfg.CallbackName = "default"
	assert.NotNil(t, GetCallbacksByConfig(cfg))
}

func TestAfterBlock(t *testing.T) {
	cfg := defaultConfig()
	filter := MockInboundFilter(cfg)
	cb := GetCallbacksByConfig(cfg)
	ctx := context.Background()
	ctx = variable.NewVariableContext(ctx)
	variable.RegisterVariable(variable.NewIndexedVariable(types.VarHeaderStatus, nil, nil, variable.BasicSetter, 0))
	ctx = variable.NewVariableContext(context.Background())
	variable.SetVariableValue(ctx, types.VarHeaderStatus, "200")

	cb.AfterBlock(filter, ctx, nil, nil, nil)
	status, err := variable.GetVariableValue(ctx, types.VarHeaderStatus)
	assert.Nil(t, err)
	assert.Equal(t, "509", status)
}

func TestDefaultCallbacks_SetConfig(t *testing.T) {
	cfg := defaultConfig()
	cb := GetCallbacksByConfig(cfg)
	assert.Empty(t, cb.GetConfig().CallbackName)
	cfg.CallbackName = "testing"
	cb = GetCallbacksByConfig(cfg)
	assert.Equal(t, "testing", cb.GetConfig().CallbackName)

	config := cb.GetConfig()
	assert.NotNil(t, config)
}
