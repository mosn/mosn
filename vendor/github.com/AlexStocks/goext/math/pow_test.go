package gxmath

import (
	"testing"
)

func TestIsPowerOf2(t *testing.T) {
	size := 64
	if !IsPowerOf2(size) {
		t.Error(size, " should be power of 2")
	}

	size = 63
	if IsPowerOf2(size) {
		t.Error(size, " should not be power of 2")
	}
}
