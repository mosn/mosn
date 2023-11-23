package otel

import (
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"mosn.io/mosn/pkg/protocol/http"
	"testing"
)

func TestHTTPHeadersCarrier(t *testing.T) {
	testKey, testValue := "Key", "Value"

	carrier := HTTPHeadersCarrier{&http.RequestHeader{
		RequestHeader: &fasthttp.RequestHeader{},
	}}
	carrier.Set(testKey, testValue)
	assert.Equal(t, testValue, carrier.Get(testKey))

	keys := carrier.Keys()
	assert.Equal(t, 1, len(keys))
	assert.Equal(t, testKey, keys[0])
}
