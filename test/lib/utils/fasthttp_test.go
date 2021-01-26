package utils

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestReadFasthttpResponseHeaders(t *testing.T) {
	resp := &fasthttp.ResponseHeader{}
	resp.Add("Test", "1")
	resp.Add("Test", "2")
	m := ReadFasthttpResponseHeaders(resp)
	if len(m["Test"]) != 2 {
		t.Fatalf("unexpected result: %v", m)
	}
}
