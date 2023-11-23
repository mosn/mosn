package otel

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTrace(t *testing.T) {
	config, err := NewTracer(nil)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	for _, configStr := range getTestData() {
		configMap, err := getConfigMap(configStr)
		assert.NoError(t, err)

		tracer, err := NewTracer(configMap)
		assert.NoError(t, err)
		assert.NotNil(t, tracer)
	}
}

func getTestData() []string {
	testCase := make([]string, 0)

	// empty
	testCase = append(testCase, "{}")
	// name only
	testCase = append(testCase, `{"service_name":"sample_sidecar"}`)
	// console
	testCase = append(testCase, `{"service_name":"sample_sidecar", "report_method":"console"}`)
	// no end point
	testCase = append(testCase, `{"service_name":"sample_sidecar", "report_method":"http"}`)
	// no end point
	testCase = append(testCase, `{"service_name":"sample_sidecar", "report_method":"grpc"}`)
	// http with endpoint
	testCase = append(testCase, `{"service_name":"sample_sidecar", "report_method":"http", "endpoint":"127.0.0.1:8080"}`)
	// grpc with endpoint
	testCase = append(testCase, `{"service_name":"sample_sidecar", "report_method":"grpc", "endpoint":"127.0.0.1:8080"}`)
	// default
	testCase = append(testCase, `{"service_name":"sample_sidecar", "report_method":"mock", "endpoint":"127.0.0.1:8080"}`)
	// custom attributes
	testCase = append(testCase, `{"service_name":"sample_sidecar", "config_map": {"key_1": "value_1", "key_2": "value_2"}}`)

	return testCase
}

func getConfigMap(jsonStr string) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonStr), &m)
	return m, err
}
