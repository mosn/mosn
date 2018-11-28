package gxsync

import (
	"reflect"
	"testing"
)

func TestMutexLayout(t *testing.T) {
	sf := reflect.TypeOf((*Mutex)(nil)).Elem().FieldByIndex([]int{0, 0})
	if sf.Name != "state" {
		t.Fatal("sync.Mutex first field should have name state")
	}
	if sf.Offset != uintptr(0) {
		t.Fatal("sync.Mutex state field should have zero offset")
	}
	if sf.Type != reflect.TypeOf(int32(1)) {
		t.Fatal("sync.Mutex state field type should be int32")
	}
}
