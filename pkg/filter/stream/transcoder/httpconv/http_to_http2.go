package httpconv

import (
	"context"
	"errors"
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

var errProtocolNotRequired = errors.New("protocol is not the required")

func init() {
	transcoder.MustRegister("httpTohttp2", &httpTohttp2{})
}

type httpTohttp2 struct{}

// Accept check the request will be transcoded or not
// http request will be translated
func (t *httpTohttp2) Accept(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) bool {
	_, ok := headers.(http.RequestHeader)
	return ok
}

// TranscodingRequest makes http request to http2 request
func (t *httpTohttp2) TranscodingRequest(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) (api.HeaderMap, api.IoBuffer, api.HeaderMap, error) {
	httpHeader, ok := headers.(http.RequestHeader)
	if !ok {
		return headers, buf, trailers, errProtocolNotRequired
	}
	// convert http header to common header, the http2 stream will encode common header to http2 header
	cheader := make(map[string]string, httpHeader.Len())
	httpHeader.VisitAll(func(key, value []byte) {
		cheader[strings.ToLower(string(key))] = string(value)
	})

	// set upstream protocol
	mosnctx.WithValue(ctx, types.ContextKeyUpStreamProtocol, string(protocol.HTTP2))

	return protocol.CommonHeader(cheader), buf, trailers, nil
}

// TranscodingResponse make http2 response to http repsonse
func (t *httpTohttp2) TranscodingResponse(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) (api.HeaderMap, api.IoBuffer, api.HeaderMap, error) {
	cheader := http2.DecodeHeader(headers)
	headerImpl := http.ResponseHeader{&fasthttp.ResponseHeader{}}
	cheader.Range(func(key, value string) bool {
		headerImpl.Set(key, value)
		return true
	})
	return headerImpl, buf, trailers, nil
}
