package gxarray

import (
	"testing"
)

func testRemoveElement(t *testing.T, array []string) {
	size := len(array)

	array, flag := RemoveElem(array, "xhello")
	if flag != false {
		t.Fatalf("return flag should be false")
	}

	array, flag = RemoveElem(array, "hello")
	if len(array) != size-1 {
		t.Fatalf("the length %d of array != %d", len(array), size-1)
	}
	if flag != true {
		t.Fatalf("return flag should be true")
	}
}

func TestRemoveElement(t *testing.T) {
	array := []string{"hello"}
	testRemoveElement(t, array)

	array = []string{"hello", "xx$$"}
	testRemoveElement(t, array)

	array = []string{"7&&", "hello", "xx$$"}
	testRemoveElement(t, array)
}
