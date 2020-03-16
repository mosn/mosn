package http2

import (
	"testing"

	"mosn.io/mosn/pkg/protocol"
)

func TestHeader(t *testing.T) {
	header := protocol.CommonHeader(make(map[string]string))
	header.Set("a", "b")
	header.Set("c", "d")
	httpHeader := EncodeHeader(header)
	if httpHeader.Get("a") != "b" {
		t.Errorf("EncodeHeader error")
	}
	if httpHeader.Get("c") != "d" {
		t.Error("EncodeHeader error")
	}
}
