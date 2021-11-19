package httpconv

import (
	"context"
	"strings"

	"github.com/valyala/fasthttp"
	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/filter/stream/transcoder"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/protocol/http2"
	"mosn.io/mosn/pkg/types"
)

func init() {
	transcoder.MustRegister("http2Tohttp", &http2Tohttp{})
}

type http2Tohttp struct{}

func (t *http2Tohttp) Accept(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) bool {
	_, ok := headers.(*http2.ReqHeader)
	return ok
}

// TranscodingRequest makes a http2 request to http request
func (t *http2Tohttp) TranscodingRequest(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) (api.HeaderMap, api.IoBuffer, api.HeaderMap, error) {
	httpHeader := http2.DecodeHeader(headers)
	headerImpl := http.RequestHeader{&fasthttp.RequestHeader{}}

	httpHeader.Range(func(key, value string) bool {
		headerImpl.Set(key, value)
		return true
	})

	mosnctx.WithValue(ctx, types.ContextKeyUpStreamProtocol, string(protocol.HTTP1))

	return headerImpl, buf, trailers, nil
}

// TranscodingResponse makes a http resposne to http2 response
func (t *http2Tohttp) TranscodingResponse(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) (api.HeaderMap, api.IoBuffer, api.HeaderMap, error) {
	httpHeader := headers.(http.ResponseHeader)
	cheader := make(map[string]string, httpHeader.Len())
	// copy headers
	httpHeader.VisitAll(func(key, value []byte) {
		cheader[strings.ToLower(string(key))] = string(value)
	})

	return protocol.CommonHeader(cheader), buf, trailers, nil
}
