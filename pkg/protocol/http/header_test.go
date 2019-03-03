package http

import (
	"testing"
	"strings"
	"github.com/valyala/fasthttp"
)

func TestRequestHeader_Add(t *testing.T) {
	header := RequestHeader{&fasthttp.RequestHeader{}, nil}
	header.Add("test-multiple", "value-one")
	header.Add("test-multiple", "value-two")

	// assert Peek results
	val := header.Peek("test-multiple")
	if string(val) != "value-one" {
		t.Errorf("RequestHeader.Get return not expected")
	}

	// assert output results
	output := header.String()
	if !strings.Contains(output, "value-one") || !strings.Contains(output, "value-two") {
		t.Errorf("RequestHeader.String not contains all header values")
	}
}


func TestResponseHeader_Add(t *testing.T) {
	header := ResponseHeader{&fasthttp.ResponseHeader{}, nil}
	header.Add("test-multiple", "value-one")
	header.Add("test-multiple", "value-two")

	// assert Peek results
	val := header.Peek("test-multiple")
	if string(val) != "value-one" {
		t.Errorf("ResponseHeader.Get return not expected")
	}

	// assert output results
	output := header.String()
	if !strings.Contains(output, "value-one") || !strings.Contains(output, "value-two") {
		t.Errorf("ResponseHeader.String not contains all header values")
	}
}

