package wasm

import (
	"context"
	"net/textproto"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
	"mosn.io/api"
	mosnhttp "mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

func Test_host_GetMethod(t *testing.T) {
	tests := []string{"GET", "POST", "OPTIONS"}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc, func(t *testing.T) {
			ctx, _ := newTestRequestContext()
			if err := variable.SetString(ctx, types.VarMethod, tc); err != nil {
				t.Fatal(err)
			}

			if want, have := tc, h.GetMethod(ctx); want != have {
				t.Errorf("unexpected method, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetMethod(t *testing.T) {
	tests := []string{"GET", "POST", "OPTIONS"}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc, func(t *testing.T) {
			ctx, _ := newTestRequestContext()

			h.SetMethod(ctx, tc)

			m, err := variable.GetString(ctx, types.VarMethod)
			if err != nil {
				t.Fatal(err)
			}

			if want, have := tc, m; want != have {
				t.Errorf("unexpected method, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_GetURI(t *testing.T) {
	tests := []struct {
		name        string
		path, query string
		expected    string
	}{
		{
			name:     "slash",
			path:     "/",
			expected: "/",
		},
		{
			name:     "space",
			path:     "/ ",
			expected: "/ ",
		},
		{
			name:     "space query",
			path:     "/ ",
			query:    "q=go+language",
			expected: "/ ?q=go+language",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestRequestContext()

			if err := variable.SetString(ctx, types.VarPath, tc.path); err != nil {
				t.Fatal(err)
			}
			if tc.query != "" {
				if err := variable.SetString(ctx, types.VarQueryString, tc.query); err != nil {
					t.Fatal(err)
				}
			}

			if want, have := tc.expected, h.GetURI(ctx); want != have {
				t.Errorf("unexpected uri, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetURI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "coerces empty path to slash",
			expected: "/",
		},
		{
			name:     "encodes space",
			input:    "/a%20b",
			expected: "/a%20b",
		},
		{
			name:     "encodes query",
			input:    "/a%20b?q=go+language",
			expected: "/a%20b?q=go+language",
		},
		{
			name:     "double slash path",
			input:    "//foo",
			expected: "//foo",
		},
		{
			name:     "empty query",
			input:    "/foo?",
			expected: "/foo?",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, r := newTestRequestContext()

			h.SetURI(ctx, tc.input)
			if want, have := tc.expected, string(r.RequestURI()); want != have {
				t.Errorf("unexpected uri, want: %v, have: %v", want, have)
			}
		})
	}
}

// Test_host_GetProtocolVersion ensures HTTP/2.0 is readable
func Test_host_GetProtocolVersion(t *testing.T) {
	tests := []string{"HTTP/1.1", "HTTP/2.0"}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc, func(t *testing.T) {
			ctx, _ := newTestRequestContext()

			if want, have := tc, h.GetProtocolVersion(ctx); want != have {
				t.Errorf("unexpected protocolVersion, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_GetRequestHeaderNames(t *testing.T) {
	ctx, _ := newTestRequestContext()

	var want []string
	for k := range testRequestHeaders {
		want = append(want, k)
	}
	sort.Strings(want)

	have := host{}.GetRequestHeaderNames(ctx)
	sort.Strings(have)
	if !reflect.DeepEqual(want, have) {
		t.Errorf("unexpected header names, want: %v, have: %v", want, have)
	}
}

func Test_host_GetRequestHeaderValues(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		expected   []string
	}{
		{
			name:       "single value",
			headerName: "Content-Type",
			expected:   []string{"text/plain"},
		},
		{
			name:       "multi-field with comma value",
			headerName: "X-Forwarded-For",
			expected:   []string{"client, proxy1", "proxy2"},
		},
		{
			name:       "empty value",
			headerName: "Empty",
			expected:   []string{""},
		},
		{
			name:       "no value",
			headerName: "Not Found",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestRequestContext()

			values := h.GetRequestHeaderValues(ctx, tc.headerName)
			if want, have := tc.expected, values; !reflect.DeepEqual(want, have) {
				t.Errorf("unexpected header values, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetRequestHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		expected   string
	}{
		{
			name:       "non-existing",
			headerName: "custom",
			expected:   "1",
		},
		{
			name:       "existing",
			headerName: "Content-Type",
			expected:   "application/json",
		},
		{
			name:       "existing lowercase",
			headerName: "content-type",
			expected:   "application/json",
		},
		{
			name:       "set to empty",
			headerName: "Custom",
		},
		{
			name:       "multi-field",
			headerName: "X-Forwarded-For",
			expected:   "proxy2",
		},
		{
			name:       "set multi-field to empty",
			headerName: "X-Forwarded-For",
			expected:   "",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestRequestContext()

			h.SetRequestHeaderValue(ctx, tc.headerName, tc.expected)
			if want, have := tc.expected, strings.Join(h.GetRequestHeaderValues(ctx, tc.headerName), "|"); want != have {
				t.Errorf("unexpected header, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_AddRequestHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		value      string
		expected   []string
	}{
		{
			name:       "non-existing",
			headerName: "new",
			value:      "1",
			expected:   []string{"1"},
		},
		{
			name:       "empty",
			headerName: "new",
			expected:   []string{""},
		},
		{
			name:       "existing",
			headerName: "X-Forwarded-For",
			value:      "proxy3",
			expected:   []string{"client, proxy1", "proxy2", "proxy3"},
		},
		{
			name:       "lowercase",
			headerName: "x-forwarded-for",
			value:      "proxy3",
			expected:   []string{"client, proxy1", "proxy2", "proxy3"},
		},
		{
			name:       "existing empty",
			headerName: "X-Forwarded-For",
			expected:   []string{"client, proxy1", "proxy2", ""},
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestRequestContext()

			h.AddRequestHeaderValue(ctx, tc.headerName, tc.value)
			if want, have := tc.expected, h.GetRequestHeaderValues(ctx, tc.headerName); !reflect.DeepEqual(want, have) {
				t.Errorf("unexpected header, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_RemoveRequestHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
	}{
		{
			name:       "doesn't exist",
			headerName: "custom",
		},
		{
			name:       "empty",
			headerName: "Empty",
		},
		{
			name:       "exists",
			headerName: "Custom",
		},
		{
			name:       "lowercase",
			headerName: "custom",
		},
		{
			name:       "multi-field",
			headerName: "X-Forwarded-For",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestRequestContext()

			h.RemoveRequestHeader(ctx, tc.headerName)
			if have := h.GetRequestHeaderValues(ctx, tc.headerName); len(have) > 0 {
				t.Errorf("unexpected headers: %v", have)
			}
		})
	}
}

func Test_host_GetResponseHeaderNames(t *testing.T) {
	ctx, _ := newTestResponseContext()

	var want []string
	for k := range testResponseHeaders {
		want = append(want, k)
	}
	sort.Strings(want)

	have := host{}.GetResponseHeaderNames(ctx)
	sort.Strings(have)
	if !reflect.DeepEqual(want, have) {
		t.Errorf("unexpected header names, want: %v, have: %v", want, have)
	}
}

func Test_host_GetResponseHeaderValues(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		expected   []string
	}{
		{
			name:       "single value",
			headerName: "Content-Type",
			expected:   []string{"text/plain"},
		},
		{
			name:       "multi-field with comma value",
			headerName: "Set-Cookie",
			expected:   []string{"a=b, c=d", "e=f"},
		},
		{
			name:       "empty value",
			headerName: "Empty",
			expected:   []string{""},
		},
		{
			name:       "no value",
			headerName: "Not Found",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestResponseContext()

			values := h.GetResponseHeaderValues(ctx, tc.headerName)
			if want, have := tc.expected, values; !reflect.DeepEqual(want, have) {
				t.Errorf("unexpected header values, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetResponseHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		expected   string
	}{
		{
			name:       "non-existing",
			headerName: "custom",
			expected:   "1",
		},
		{
			name:       "existing",
			headerName: "Content-Type",
			expected:   "application/json",
		},
		{
			name:       "existing lowercase",
			headerName: "content-type",
			expected:   "application/json",
		},
		{
			name:       "set to empty",
			headerName: "Custom",
		},
		{
			name:       "multi-field",
			headerName: "Set-Cookie",
			expected:   "proxy2",
		},
		{
			name:       "set multi-field to empty",
			headerName: "Set-Cookie",
			expected:   "",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestResponseContext()

			h.SetResponseHeaderValue(ctx, tc.headerName, tc.expected)
			if want, have := tc.expected, strings.Join(h.GetResponseHeaderValues(ctx, tc.headerName), "|"); want != have {
				t.Errorf("unexpected header, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_AddResponseHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		value      string
		expected   []string
	}{
		{
			name:       "non-existing",
			headerName: "new",
			value:      "1",
			expected:   []string{"1"},
		},
		{
			name:       "empty",
			headerName: "new",
			expected:   []string{""},
		},
		{
			name:       "existing",
			headerName: "Set-Cookie",
			value:      "g=h",
			expected:   []string{"a=b, c=d", "e=f", "g=h"},
		},
		{
			name:       "lowercase",
			headerName: "set-Cookie",
			value:      "g=h",
			expected:   []string{"a=b, c=d", "e=f", "g=h"},
		},
		{
			name:       "existing empty",
			headerName: "Set-Cookie",
			value:      "",
			expected:   []string{"a=b, c=d", "e=f", ""},
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestResponseContext()

			h.AddResponseHeaderValue(ctx, tc.headerName, tc.value)
			if want, have := tc.expected, h.GetResponseHeaderValues(ctx, tc.headerName); !reflect.DeepEqual(want, have) {
				t.Errorf("unexpected header, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_RemoveResponseHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
	}{
		{
			name:       "doesn't exist",
			headerName: "new",
		},
		{
			name:       "empty",
			headerName: "Empty",
		},
		{
			name:       "exists",
			headerName: "Custom",
		},
		{
			name:       "lowercase",
			headerName: "custom",
		},
		{
			name:       "multi-field",
			headerName: "Set-Cookie",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestResponseContext()

			h.RemoveResponseHeader(ctx, tc.headerName)
			if have := h.GetResponseHeaderValues(ctx, tc.headerName); len(have) > 0 {
				t.Errorf("unexpected headers: %v", have)
			}
		})
	}
}

func newTestRequestContext() (ctx context.Context, r mosnhttp.RequestHeader) {
	r = newRequestHeader()
	setTestRequestHeaders(r)
	ctx = variable.NewVariableContext(context.Background())
	ctx = context.WithValue(ctx, filterKey{}, &filter{reqHeaders: r})
	return
}

func newTestResponseContext() (ctx context.Context, r mosnhttp.ResponseHeader) {
	r = newResponseHeader()
	setTestResponseHeaders(r)
	ctx = variable.NewVariableContext(context.Background())
	ctx = context.WithValue(ctx, filterKey{}, &filter{respHeaders: r})
	return
}

func newRequestHeader() mosnhttp.RequestHeader {
	return mosnhttp.RequestHeader{RequestHeader: &fasthttp.RequestHeader{}}
}

func newResponseHeader() mosnhttp.ResponseHeader {
	return mosnhttp.ResponseHeader{ResponseHeader: &fasthttp.ResponseHeader{}}
}

func setTestRequestHeaders(h api.HeaderMap) {
	setTestHeaders(h, testRequestHeaders)
}

func setTestResponseHeaders(h api.HeaderMap) {
	setTestHeaders(h, testResponseHeaders)
}

func setTestHeaders(h api.HeaderMap, t map[string][]string) {
	for k, vs := range t {
		h.Set(k, vs[0])
		for _, v := range vs[1:] {
			h.Add(k, v)
		}
	}
}

func headerValues(h api.HeaderMap, name string) (values []string) {
	name = textproto.CanonicalMIMEHeaderKey(name)
	h.Range(func(key, value string) bool {
		if key == name {
			values = append(values, value)
		}
		return true
	})
	return values
}

// Note: senders are supposed to concatenate multiple fields with the same
// name on comma, except the response header Set-Cookie. That said, a lot
// of middleware don't know about this and may repeat other headers anyway.
// See https://www.rfc-editoreqHeaders.org/rfc/rfc9110#section-5.2

var (
	testRequestHeaders = map[string][]string{
		"Content-Type":    {"text/plain"},
		"Custom":          {"1"},
		"X-Forwarded-For": {"client, proxy1", "proxy2"},
		"Empty":           {""},
	}
	testResponseHeaders = map[string][]string{
		"Content-Type": {"text/plain"},
		"Custom":       {"1"},
		"Set-Cookie":   {"a=b, c=d", "e=f"},
		"Empty":        {""},
	}
)
