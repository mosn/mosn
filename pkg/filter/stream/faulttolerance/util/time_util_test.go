package util

import (
	"testing"
)

func Test_time(t *testing.T) {
	if GetNowMS() >= 0 {
		t.Log("Test_time Succeed")
	} else {
		t.Error("Test_time Failed")
	}
}
