package utils

import "github.com/valyala/fasthttp"

func ReadFasthttpResponseHeaders(h *fasthttp.ResponseHeader) map[string][]string {
	m := map[string][]string{}
	h.VisitAll(func(key, value []byte) {
		k := string(key)
		s := m[k]
		s = append(s, string(value))
		m[k] = s
	})
	return m
}
