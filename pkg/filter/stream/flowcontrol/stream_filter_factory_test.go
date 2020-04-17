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
