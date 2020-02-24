package http

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/valyala/fasthttp"
	mosnhttp "mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

func BuildHTTP1Request(method string, header map[string]string, body []byte) (types.HeaderMap, types.IoBuffer) {
	// makes a mosnhttp.RequestHeader
	fh := &fasthttp.RequestHeader{}
	fh.SetMethod(method)
	h := mosnhttp.RequestHeader{
		RequestHeader: fh,
	}
	for k, v := range header {
		h.Set(k, v)
	}
	if len(body) > 0 {
		buf := buffer.NewIoBufferBytes(body)
		return h, buf
	}
	return h, nil
}

type VerifyConfig struct {
	ExpectedStatus int
	// if ExepctedHeader is nil, means do not care about header
	// if ExepctedHeader is exists, means resposne Header should contain all the ExpectedHeader
	// TODO :  support regex
	ExpectedHeader map[string]string
	// if ExpectedBody is nil, means do not care about body
	// TODO: support regex
	ExpectedBody []byte
	// if ExpectedRT is zero, means do not care about rt
	// if ExpectedRT is not zero, means response's rt should no more than it
	ExpectedRT time.Duration
	// if MinRT is zero means do not care about it
	// if MinRT is not zero, means response's rt should more than it
	MinRT time.Duration
}

func (cfg *VerifyConfig) Verify(resp *Response) bool {
	if resp.Header.StatusCode() != cfg.ExpectedStatus {
		fmt.Printf("expected receive status %d, but got %d\n", cfg.ExpectedStatus, resp.Header.StatusCode())
		return false
	}
	for k, v := range cfg.ExpectedHeader {
		if value, ok := resp.Header.Get(k); !ok || value != v {
			fmt.Printf("expected receive header %v, but got %v\n", cfg.ExpectedHeader, resp.Header)
			return false
		}
	}
	// if ExpectedBody is not nil, but length is zero, means expected empty content
	if cfg.ExpectedBody != nil {
		if !bytes.Equal(cfg.ExpectedBody, resp.Data) {
			fmt.Printf("expected receive header %s, but got %s\n", string(cfg.ExpectedBody), string(resp.Data))
			return false
		}
	}
	if cfg.ExpectedRT > 0 {
		if resp.Cost > cfg.ExpectedRT {
			fmt.Printf("expected receive rt is %v, but cost %v\n", cfg.ExpectedRT, resp.Cost)
			return false
		}
	}
	if cfg.MinRT > 0 {
		if resp.Cost < cfg.MinRT {
			fmt.Printf("expected receive rt at least %v, but cost %v\n", cfg.MinRT, resp.Cost)
			return false
		}
	}
	return true
}

var DefaultVeirfy = &VerifyConfig{
	ExpectedStatus: http.StatusOK,
}
