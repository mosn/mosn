package flowcontrol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamFilterFactory(t *testing.T) {
	mockConfig := `{
    "global_switch": false,
    "monitor": false,
    "limit_key_type": "PATH",
    "action": {
        "status": 0,
        "body": ""
    },
    "rules": [
        {
            "resource": "/http",
            "limitApp": "",
            "grade": 1,
            "count": 1,
            "strategy": 0
        }
    ]
}`
	data := map[string]interface{}{}
	err := json.Unmarshal([]byte(mockConfig), &data)
	assert.Nil(t, err)
	f, err := createRpcFlowControlFilterFactory(data)
	assert.NotNil(t, f)
	assert.Nil(t, err)
}

func TestIsValidCfg(t *testing.T) {
	validConfig := `{
    "global_switch": false,
    "monitor": false,
    "limit_key_type": "PATH",
    "action": {
        "status": 0,
        "body": ""
    },
    "rules": [
        {
            "resource": "/http",
            "limitApp": "",
            "grade": 1,
            "count": 1,
            "strategy": 0
        }
    ]
}`
	cfg := &Config{}
	err := json.Unmarshal([]byte(validConfig), cfg)
	assert.Nil(t, err)
	isValid, err := isValidConfig(cfg)
	assert.True(t, isValid)
	assert.Nil(t, err)

	// invalid
	invalidCfg := `{
    "global_switch": false,
    "monitor": false,
    "limit_key_type": "????",
    "action": {
        "status": 0,
        "body": ""
    },
    "rules": [
        {
            "resource": "/http",
            "limitApp": "",
            "grade": 1,
            "count": 1,
            "strategy": 0
        }
    ]
}`
	err = json.Unmarshal([]byte(invalidCfg), cfg)
	assert.Nil(t, err)
	isValid, err = isValidConfig(cfg)
	assert.False(t, isValid)
	assert.NotNil(t, err)
}
