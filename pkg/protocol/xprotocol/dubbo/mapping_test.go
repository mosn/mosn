package dubbo

import (
	"reflect"
	"testing"
)

func TestDubboStatusMapping_MappingHeaderStatusCode(t *testing.T) {
	dubboStatusMapping := &dubboStatusMapping{}
	code, err := dubboStatusMapping.MappingHeaderStatusCode(nil, nil)
	equal(t, code, 0)
	equal(t, err != nil, true)

	frame := &Frame{
		Header: Header{
			Status: 0,
		},
	}
	code, err = dubboStatusMapping.MappingHeaderStatusCode(nil, frame)
	equal(t, code, 500)
	equal(t, err == nil, true)

	frame.Header.Status = 20
	code, err = dubboStatusMapping.MappingHeaderStatusCode(nil, frame)
	equal(t, code, 200)
	equal(t, err == nil, true)

	frame.Header.Status = 30
	code, err = dubboStatusMapping.MappingHeaderStatusCode(nil, frame)
	equal(t, code, 504)
	equal(t, err == nil, true)

	frame.Header.Status = 31
	code, err = dubboStatusMapping.MappingHeaderStatusCode(nil, frame)
	equal(t, code, 504)
	equal(t, err == nil, true)

	frame.Header.Status = 40
	code, err = dubboStatusMapping.MappingHeaderStatusCode(nil, frame)
	equal(t, code, 504)
	equal(t, err == nil, true)

	frame.Header.Status = 60
	code, err = dubboStatusMapping.MappingHeaderStatusCode(nil, frame)
	equal(t, code, 502)
	equal(t, err == nil, true)

	frame.Header.Status = 100
	code, err = dubboStatusMapping.MappingHeaderStatusCode(nil, frame)
	equal(t, code, 507)
	equal(t, err == nil, true)
}

func equal(t *testing.T, got interface{}, want interface{}) {
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s, want %s", got, want)
	}
}
