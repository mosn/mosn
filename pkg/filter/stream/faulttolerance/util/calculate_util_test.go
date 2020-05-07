package util

import (
	"testing"
)

func Test_Calculate(t *testing.T) {
	if DivideInt64(3, 12) != 0.25 {
		t.Error("Test_Calculate failed")
	}
	if DivideInt64(1, 3) != 0.33 {
		t.Error("Test_Calculate failed")
	}
	if DivideFloat64(3.0, 12.0) != 0.25 {
		t.Error("Test_Calculate failed")
	}
	if DivideFloat64(1.0166, 3.0456) != 0.33 {
		t.Error("Test_Calculate failed")
	}
}
