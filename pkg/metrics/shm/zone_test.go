package shm

import (
	"testing"
	"strconv"
)

func TestNewSharedMetrics(t *testing.T) {
	zone := InitMetricsZone("TestNewSharedMetrics", 10 * 1024 * 1024)
	defer zone.Detach()

	entryCount := 1000
	for i := 0; i < entryCount; i++ {
		entry, err := defaultZone.alloc("testEntry" + strconv.Itoa(i))
		if err != nil {
			t.Error(err)
		}

		entry.value = int64(i + 1)
	}

	// re-alloc and re-access
	entry, err := defaultZone.alloc("testEntry0")
	if err != nil {
		t.Error(err)
	}

	if entry.value != 1 {
		t.Error("testEntry0 value not correct")
	}
}
