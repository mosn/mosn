package jwtauthn

import (
	"context"
	"testing"

	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/pkg/variable"

	_ "mosn.io/mosn/pkg/stream/http"
	"mosn.io/mosn/pkg/types"
)

func TestOnReceive(t *testing.T) {
	exampleConfig := `
{
	"providers": {
        "origins-0": {
            "issuer": "testing@secure.istio.io",
            "localJwks": {
                "inlineString": "{ \"keys\":[ {\"e\":\"AQAB\",\"kid\":\"DHFbpoIUqrY8t2zpA2qXfCmr5VO5ZEr4RzHU_-envvQ\",\"kty\":\"RSA\",\"n\":\"xAE7eB6qugXyCAG3yhh7pkDkT65pHymX-P7KfIupjf59vsdo91bSP9C8H07pSAGQO1MV_xFj9VswgsCg4R6otmg5PV2He95lZdHtOcU5DXIg_pbhLdKXbi66GlVeK6ABZOUW3WYtnNHD-91gVuoeJT_DwtGGcp4ignkgXfkiEm4sw-4sfb4qdt5oLbyVpmW6x9cfa7vs2WTfURiCrBoUqgBo_-4WTiULmmHSGZHOjzwa8WtrtOQGsAFjIbno85jp6MnGGGZPYZbDAa_b3y5u-YpW7ypZrvD8BgtKVjgtQgZhLAGezMt0ua3DRrWnKqTZ0BJ_EyxOGuHJrLsn00fnMQ\"}]}"
            },
            "payloadInMetadata": "testing@secure.istio.io"
        }
    },
    "rules": [
        {
            "match": {
                "prefix": "/"
            },
            "requires": {
                "requiresAny": {
                    "requirements": [
                        {
                            "providerName": "origins-0"
                        },
                        {
                            "allowMissing": {}
                        }
                    ]
                }
            }
        }
    ]
}
`

	var config jwtauthnv3.JwtAuthentication
	if err := jsonpb.UnmarshalString(exampleConfig, &config); err != nil {
		t.Error(err)
		t.FailNow()
	}

	filterConfig, err := NewFilterConfig(&config)
	assert.Nil(t, err)
	filter := newJwtAuthnFilter(filterConfig)

	tests := []struct {
		name       string
		headers    api.HeaderMap
		hijackCode int
		status     api.StreamFilterStatus
	}{
		{
			name:       "invalid token",
			headers:    newHeaders([2]string{"Authorization", "Bearer invalidToken"}),
			hijackCode: 401,
			status:     api.StreamFilterStop,
		},
		{
			name:       "allow missing",
			headers:    newHeaders(),
			hijackCode: 200,
			status:     api.StreamFilterContinue,
		},
		{
			name:       "good token",
			headers:    newHeaders([2]string{"Authorization", "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcXJZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJleHAiOjQ2ODU5ODk3MDAsImZvbyI6ImJhciIsImlhdCI6MTUzMjM4OTcwMCwiaXNzIjoidGVzdGluZ0BzZWN1cmUuaXN0aW8uaW8iLCJzdWIiOiJ0ZXN0aW5nQHNlY3VyZS5pc3Rpby5pbyJ9.CfNnxWP2tcnR9q0vxyxweaF3ovQYHYZl82hAUsn21bwQd9zP7c-LS9qd_vpdLG4Tn1A15NxfCjp5f7QNBUo-KC9PJqYpgGbaXhaGx7bEdFWjcwv3nZzvc7M__ZpaCERdwU7igUmJqYGBYQ51vr2njU9ZimyKkfDe3axcyiBZde7G6dabliUosJvvKOPcKIWPccCgefSj_GNfwIip3-SsFdlR7BtbVUcqR-yv-XOxJ3Uc1MI0tz3uMiiZcyPV7sNCU4KRnemRIMHVOfuvHsU60_GhGbiSFzgPTAa9WTltbnarTbxudb_YEOx12JiwYToeX0DCPb43W1tzIBxgm8NxUg"}),
			hijackCode: 200,
			status:     api.StreamFilterContinue,
		},
	}

	v := variable.NewStringVariable(types.VarHttpRequestPath, nil, func(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
		return "/", nil
	}, nil, 0)
	variable.Register(v)

	ctx := variable.NewVariableContext(context.Background())

	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			handler := &mockStreamReceiverFilterCallbacks{
				hijackCode: 200,
			}
			filter.SetReceiveFilterHandler(handler)
			status := filter.OnReceive(ctx, tc.headers, nil, nil)
			assert.Equal(t, tc.status, status)
			assert.Equal(t, tc.hijackCode, handler.hijackCode)
		})
	}
}
