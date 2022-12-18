package keyauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"mosn.io/api"
	pb "mosn.io/mosn/pkg/filter/stream/auth/keyauth/keyauthpb"
	"mosn.io/mosn/pkg/protocol/http"
)

func TestAuthenticate(t *testing.T) {

	config := pb.Authenticator{
		FromHeader: "authKey",
		FromParam:  "authKey",
		Keys:       []string{"key1"},
	}
	auth := NewKeyAuth(&config)

	assert.True(t, auth.Authenticate(newHeaders(map[string]string{"authKey": "key1"}), ""))
	assert.True(t, auth.Authenticate(newHeaders(map[string]string{}), "authKey=key1"))
	assert.False(t, auth.Authenticate(newHeaders(map[string]string{"auth": "key1"}), ""))
	assert.False(t, auth.Authenticate(newHeaders(map[string]string{}), "auth=key1"))
	assert.False(t, auth.Authenticate(newHeaders(map[string]string{"authKey": "errorKey"}), ""))
	assert.False(t, auth.Authenticate(newHeaders(map[string]string{}), "authKey=errorKey"))
	assert.False(t, auth.Authenticate(newHeaders(map[string]string{}), ""))
}

func newHeaders(headers map[string]string) api.HeaderMap {
	res := &http.RequestHeader{
		RequestHeader: &fasthttp.RequestHeader{},
	}
	for k, v := range headers {
		res.Set(k, v)
	}
	return res
}
