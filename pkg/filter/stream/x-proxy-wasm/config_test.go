package x_proxy_wasm

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestXProxyWasmConfigFromGlobalPlugin(t *testing.T) {
	configMap := map[string]interface{}{
		"from_wasm_plugin": "global_plugin",
		"instance_num":     2,
		"root_context_id":  1,
		"user_config1":     "user_value1",
		"user_config2":     "user_value2",
	}

	config, err := parseFilterConfig(configMap)
	assert.Nil(t, err)
	assert.Nil(t, config.VmConfig)
	assert.Equal(t, config.InstanceNum, 0)
	assert.Equal(t, len(config.UserData), 2)
}

func TestXProxyWasmConfigWithoutUserData(t *testing.T) {
	configMap := map[string]interface{}{
		"from_wasm_plugin": "global_plugin",
		"instance_num":     2,
		"root_context_id":  1,
	}

	config, err := parseFilterConfig(configMap)
	assert.Nil(t, err)
	assert.Nil(t, config.VmConfig)
	assert.Equal(t, config.InstanceNum, 0)
	assert.Equal(t, len(config.UserData), 0)
}

func TestXProxyWasmConfigWithVM(t *testing.T) {
	configMap := map[string]interface{}{
		"vm_config": map[string]interface{}{
			"engine": "wasmer",
			"path":   "path",
			"cpu":    100,
		},
		"instance_num":    2,
		"root_context_id": 1,
		"user_config1":    "user_value1",
		"user_config2":    "user_value2",
		"user_config3":    "user_value3",
		"user_config4":    "user_value4",
	}

	config, err := parseFilterConfig(configMap)
	assert.Nil(t, err)
	assert.Equal(t, config.FromWasmPlugin, "")
	assert.NotNil(t, config.VmConfig)
	assert.Equal(t, config.InstanceNum, 2)
	assert.Equal(t, len(config.UserData), 4)
}
