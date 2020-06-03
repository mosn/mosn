package http2

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http2"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
)

func Test_get_prefixProtocolVar(t *testing.T) {
	headerName := "Header_key"
	expect := "header_value"
	headers := http2.NewHeaderMap(http.Header(map[string][]string{}))
	headers.Set(headerName, expect)

	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyDownStreamHeaders, headers)

	ctx = mosnctx.WithValue(ctx, types.ContextKeyDownStreamProtocol, protocol.HTTP2)

	actual, err := variable.GetProtocolResource(ctx, api.HEADER, headerName)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if !assert.Equalf(t, expect, actual, "header value expect to be %s, but get %s") {
		t.FailNow()
	}
}
