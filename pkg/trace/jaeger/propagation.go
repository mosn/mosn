package jaeger

import "mosn.io/mosn/pkg/protocol/http"

//HTTPHeadersCarrier
type HTTPHeadersCarrier http.RequestHeader

// Set conforms to the HTTPHeadersCarrier interface.
func (c HTTPHeadersCarrier) Set(key, val string) {
	h := http.RequestHeader(c)
	h.Set(key, val)
}

// ForeachKey conforms to the HTTPHeadersCarrier interface.
func (c HTTPHeadersCarrier) ForeachKey(handler func(key, val string) error) error {
	h := http.RequestHeader(c)

	stopped := false
	h.VisitAll(func(key, value []byte) {
		if stopped {
			return
		}

		if handler(string(key), string(value)) != nil {
			stopped = true
		}
	})
	return nil
}
