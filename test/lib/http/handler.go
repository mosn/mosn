package http

import (
	"net/http"
	"reflect"
	"strings"
)

type ResponseBuilder struct {
	StatusCode int
	Header     map[string]string
	Body       []byte
}

func (b *ResponseBuilder) Build(w http.ResponseWriter) (int, error) {
	for k, v := range b.Header {
		w.Header().Set(k, v)
	}
	// WriteHeader should be called after Header.Set
	w.WriteHeader(b.StatusCode)
	return w.Write(b.Body)
}

var DefaultBuilder = &ResponseBuilder{
	StatusCode: http.StatusOK,
	Header: map[string]string{
		"mosn-test-default": "http1",
	},
	Body: []byte("default-http1"),
}

var DefaultErrorBuilder = &ResponseBuilder{
	StatusCode: http.StatusInternalServerError, // 500
	Header: map[string]string{
		"error-mesage": "no matched config",
	},
}

// TODO: support more
type HTTPResonseConfig struct {
	ExpectedHeader      http.Header // map[string][]string
	UnexpectedHeaderKey []string
	ExpectedMethod      string
	//
	Builder      *ResponseBuilder
	ErrorBuidler *ResponseBuilder
}

func (cfg *HTTPResonseConfig) Match(r *http.Request) bool {
	// Verify Request
	if cfg.ExpectedMethod != "" { // empty ExpectedMethod means accept any method
		if r.Method != cfg.ExpectedMethod {
			return false
		}
	}
	header := r.Header
	if len(header) < len(cfg.ExpectedHeader) {
		return false
	}
	for key, value := range cfg.ExpectedHeader {
		// needs to verify the string slice sequence
		key = strings.Title(key)
		if !reflect.DeepEqual(header[key], value) {
			return false
		}
	}
	for _, key := range cfg.UnexpectedHeaderKey {
		if _, ok := header[key]; ok {
			return false
		}
	}
	return true
}

func (cfg *HTTPResonseConfig) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if cfg.Match(r) {
		cfg.Builder.Build(w)
	} else {
		cfg.ErrorBuidler.Build(w)
	}
}

type HTTPServe struct {
	Configs map[string]*HTTPResonseConfig
}

func (s *HTTPServe) Serve(srv *MockServer) {
	for pattern, cfg := range s.Configs {
		srv.HandleFunc(pattern, cfg.ServeHTTP)
	}
}

var DefaultHTTPServe = &HTTPServe{
	Configs: map[string]*HTTPResonseConfig{
		"/": &HTTPResonseConfig{
			Builder:      DefaultBuilder,
			ErrorBuidler: DefaultErrorBuilder,
		},
	},
}
