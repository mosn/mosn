package wasm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mosn.io/pkg/buffer"
	"net/textproto"
	"net/url"

	"github.com/valyala/fasthttp"
	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/protocol/http"
	"mosn.io/pkg/variable"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
)

var _ handler.Host = host{}

type host struct{}

// EnableFeatures implements the same method as documented on handler.Host.
func (host) EnableFeatures(ctx context.Context, features handler.Features) handler.Features {
	// Remove trailers until it is supported. See httpwasm/httpwasm#2145
	features = features &^ handler.FeatureTrailers
	if s, ok := ctx.Value(filterKey{}).(*filter); ok {
		s.enableFeatures(features)
	}
	return features
}

func (host) GetMethod(ctx context.Context) (method string) {
	return mustGetString(ctx, types.VarMethod)
}

func (host) SetMethod(ctx context.Context, method string) {
	mustSetString(ctx, types.VarMethod, method)
}

func (host) GetProtocolVersion(ctx context.Context) string {
	p := mustGetString(ctx, types.VarProtocol)
	switch p {
	case "Http1":
		return "HTTP/1.1"
	case "Http2":
		return "HTTP/2.0"
	}
	return p
}

func (host) GetRequestHeaderNames(ctx context.Context) (names []string) {
	return getHeaderNames(filterFromContext(ctx).reqHeaders)
}

func (host) GetRequestHeaderValues(ctx context.Context, name string) []string {
	return getHeaders(filterFromContext(ctx).reqHeaders, name)
}

func (h host) SetRequestHeaderValue(ctx context.Context, name, value string) {
	f := filterFromContext(ctx)
	if hdr := f.reqHeaders; hdr == nil {
		f.reqHeaders = http.RequestHeader{RequestHeader: &fasthttp.RequestHeader{}}
		h.SetRequestHeaderValue(ctx, name, value)
	} else if req, ok := hdr.(http.RequestHeader); ok {
		removeHeader(req, name) // set doesn't currently overwrite all fields!
		req.Set(name, value)
	} else {
		panic(fmt.Errorf("BUG: unknown type of request header: %v", h))
	}
}

func (h host) AddRequestHeaderValue(ctx context.Context, name, value string) {
	f := filterFromContext(ctx)
	if hdr := f.reqHeaders; hdr == nil {
		h.SetRequestHeaderValue(ctx, name, value)
	} else if req, ok := hdr.(http.RequestHeader); ok {
		req.Add(name, value)
	} else {
		panic(fmt.Errorf("BUG: unknown type of request header: %v", hdr))
	}
}

func (host) RemoveRequestHeader(ctx context.Context, name string) {
	f := filterFromContext(ctx)
	if hdr := f.reqHeaders; hdr == nil {
		// noop
	} else if req, ok := hdr.(http.RequestHeader); ok {
		removeHeader(req, name)
	} else {
		panic(fmt.Errorf("BUG: unknown type of request header: %v", hdr))
	}
}

func removeHeader(h api.HeaderMap, name string) {
	name = textproto.CanonicalMIMEHeaderKey(name)
	h.Range(func(key, value string) bool {
		if key == name {
			h.Del(key)
		}
		return true
	})
}

func (host) RequestBodyReader(ctx context.Context) io.ReadCloser {
	reqBody := filterFromContext(ctx).reqBody
	if reqBody == nil {
		return io.NopCloser(buffer.NewIoBufferEOF())
	}
	b := filterFromContext(ctx).reqBody.Bytes()
	return io.NopCloser(bytes.NewReader(b))
}

func (host) RequestBodyWriter(ctx context.Context) io.Writer {
	f := filterFromContext(ctx)
	f.reqBody.Reset()
	return writerFunc(f.WriteRequestBody)
}

func (host) GetRequestTrailerNames(ctx context.Context) (names []string) {
	return // no-op because trailers are unsupported: httpwasm/httpwasm#2145
}

func (host) GetRequestTrailerValues(ctx context.Context, name string) (values []string) {
	return // no-op because trailers are unsupported: httpwasm/httpwasm#2145
}

func (host) AddRequestTrailerValue(ctx context.Context, name, value string) {
	// panic because the user should know that trailers are not supported.
	panic("trailers unsupported: httpwasm/httpwasm#2145")
}

func (host) RemoveRequestTrailer(ctx context.Context, name string) {
	// panic because the user should know that trailers are not supported.
	panic("trailers unsupported: httpwasm/httpwasm#2145")
}

func (host) SetRequestTrailerValue(ctx context.Context, name, value string) {
	// panic because the user should know that trailers are not supported.
	panic("trailers unsupported: httpwasm/httpwasm#2145")
}

func (host) GetURI(ctx context.Context) string {
	p, err := variable.GetString(ctx, types.VarPathOriginal)
	if err != nil {
		p, _ = variable.GetString(ctx, types.VarPath)
	}
	q, qErr := variable.GetString(ctx, types.VarQueryString)
	if qErr != nil {
		// No query, so an error is returned.
		return p
	}
	return fmt.Sprintf("%s?%s", p, q)
}

func (host) SetURI(ctx context.Context, uri string) {
	req := filterFromContext(ctx).reqHeaders.(http.RequestHeader)
	if uri == "" { // url.ParseRequestURI fails on empty
		req.SetRequestURI("/")
		mustSetString(ctx, types.VarPathOriginal, "/")
		mustSetString(ctx, types.VarPath, "/")
		mustSetString(ctx, types.VarQueryString, "")
		return
	}
	req.SetRequestURI(uri)
	u, err := url.ParseRequestURI(uri)
	if err != nil {
		panic(err)
	}
	mustSetString(ctx, types.VarPathOriginal, u.EscapedPath())
	mustSetString(ctx, types.VarPath, u.Path)
	if len(u.RawQuery) > 0 || u.ForceQuery {
		mustSetString(ctx, types.VarQueryString, u.RawQuery)
	}
}

func (host) GetStatusCode(ctx context.Context) uint32 {
	f := filterFromContext(ctx)
	if resp, ok := f.respHeaders.(http.ResponseHeader); ok {
		return uint32(resp.StatusCode())
	} else {
		return uint32(f.statusCode)
	}
}

func (host) SetStatusCode(ctx context.Context, statusCode uint32) {
	f := filterFromContext(ctx)
	if resp, ok := f.respHeaders.(http.ResponseHeader); ok {
		resp.SetStatusCode(int(statusCode))
	} else {
		f.statusCode = int(statusCode)
	}
}

func (host) GetResponseHeaderNames(ctx context.Context) (names []string) {
	return getHeaderNames(filterFromContext(ctx).respHeaders)
}

func (host) GetResponseHeaderValues(ctx context.Context, name string) []string {
	return getHeaders(filterFromContext(ctx).respHeaders, name)
}

func (h host) SetResponseHeaderValue(ctx context.Context, name, value string) {
	f := filterFromContext(ctx)
	if hdr := f.respHeaders; hdr == nil {
		f.respHeaders = http.ResponseHeader{ResponseHeader: &fasthttp.ResponseHeader{}}
		h.SetResponseHeaderValue(ctx, name, value)
	} else if resp, ok := hdr.(http.ResponseHeader); ok {
		removeHeader(resp, name) // set doesn't currently overwrite all fields!
		resp.Set(name, value)
	} else {
		panic(fmt.Errorf("BUG: unknown type of response header: %v", h))
	}
}

func (h host) AddResponseHeaderValue(ctx context.Context, name, value string) {
	f := filterFromContext(ctx)
	if hdr := f.respHeaders; hdr == nil {
		h.SetRequestHeaderValue(ctx, name, value)
	} else if resp, ok := hdr.(http.ResponseHeader); ok {
		resp.Add(name, value)
	} else {
		panic(fmt.Errorf("BUG: unknown type of response header: %v", hdr))
	}
}

func (host) RemoveResponseHeader(ctx context.Context, name string) {
	f := filterFromContext(ctx)
	if hdr := f.respHeaders; hdr == nil {
		// noop
	} else if resp, ok := hdr.(http.ResponseHeader); ok {
		removeHeader(resp, name)
	} else {
		panic(fmt.Errorf("BUG: unknown type of response header: %v", hdr))
	}
}

func (host) ResponseBodyReader(ctx context.Context) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(filterFromContext(ctx).respBody))
}

func (host) ResponseBodyWriter(ctx context.Context) io.Writer {
	f := filterFromContext(ctx)
	f.respBody = nil // reset
	return writerFunc(f.WriteResponseBody)
}

func (host) GetResponseTrailerNames(ctx context.Context) (names []string) {
	return // no-op because trailers are unsupported: httpwasm/httpwasm#2145
}

func (host) GetResponseTrailerValues(ctx context.Context, name string) (values []string) {
	return // no-op because trailers are unsupported: httpwasm/httpwasm#2145
}

func (host) SetResponseTrailerValue(ctx context.Context, name, value string) {
	// panic because the user should know that trailers are not supported.
	panic("trailers unsupported: httpwasm/httpwasm#2145")
}

func (host) AddResponseTrailerValue(ctx context.Context, name, value string) {
	// panic because the user should know that trailers are not supported.
	panic("trailers unsupported: httpwasm/httpwasm#2145")
}

func (host) RemoveResponseTrailer(ctx context.Context, name string) {
	// panic because the user should know that trailers are not supported.
	panic("trailers unsupported: httpwasm/httpwasm#2145")
}

func mustGetString(ctx context.Context, name string) string {
	if s, err := variable.GetString(ctx, name); err != nil {
		panic(err)
	} else {
		return s
	}
}

func mustSetString(ctx context.Context, name, value string) {
	if err := variable.SetString(ctx, name, value); err != nil {
		panic(err)
	}
}

func getHeaderNames(headers api.HeaderMap) (names []string) {
	if headers == nil {
		return
	}

	// headers.Range can return the same name multiple times, so de-dupe
	var headerNames map[string]struct{}

	headers.Range(func(key, value string) bool {
		if headerNames == nil {
			headerNames = map[string]struct{}{key: {}}
		} else if _, ok := headerNames[key]; ok {
			return true // dupe
		} else {
			headerNames[key] = struct{}{}
		}
		names = append(names, key)
		return true
	})
	return
}

func getHeaders(headers api.HeaderMap, name string) (values []string) {
	if headers == nil {
		return
	}
	name = textproto.CanonicalMIMEHeaderKey(name)
	headers.Range(func(key, value string) bool {
		if key == name && value != http.PlaceHolder {
			values = append(values, value)
		}
		return true
	})
	return
}
