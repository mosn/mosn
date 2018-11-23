package sofarpc

import (
	"reflect"
	"testing"
)

func TestReqCommandClone(t *testing.T) {
	headers := map[string]string{"service": "test"}
	// test headers, the others ignore
	reqCmd := &BoltRequestCommand{
		RequestHeader: headers,
	}
	reqClone := reqCmd.Clone()
	if !reflect.DeepEqual(reqCmd, reqClone) {
		t.Error("clone data not equal")
	}
	// modify clone header
	reqClone.Set("testclone", "test")
	reqClone.Set("service", "clone")
	// new header is setted
	if v, ok := reqClone.Get("service"); !ok || v != "clone" {
		t.Error("clone header is not setted")
	}
	if v, ok := reqClone.Get("testclone"); !ok || v != "test" {
		t.Error("clone header is not setted")
	}
	// original header is not effected
	if v, ok := reqCmd.Get("service"); !ok || v != "test" {
		t.Error("original header is chaned")
	}
	if _, ok := reqCmd.Get("testclone"); ok {
		t.Error("original header is chaned")
	}
}

func TestRespCommandClone(t *testing.T) {
	headers := map[string]string{"service": "test"}
	// test headers, the others ignore
	respCmd := &BoltResponseCommand{
		ResponseHeader: headers,
	}
	respClone := respCmd.Clone()
	if !reflect.DeepEqual(respCmd, respClone) {
		t.Error("clone data not equal")
	}
	// modify clone header
	respClone.Set("testclone", "test")
	respClone.Set("service", "clone")
	// new header is setted
	if v, ok := respClone.Get("service"); !ok || v != "clone" {
		t.Error("clone header is not setted")
	}
	if v, ok := respClone.Get("testclone"); !ok || v != "test" {
		t.Error("clone header is not setted")
	}
	// original header is not effected
	if v, ok := respCmd.Get("service"); !ok || v != "test" {
		t.Error("original header is chaned")
	}
	if _, ok := respCmd.Get("testclone"); ok {
		t.Error("original header is chaned")
	}
}
