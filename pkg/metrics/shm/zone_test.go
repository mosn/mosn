package shm

import (
	"testing"
	"strconv"
)

func TestNewSharedMetrics(t *testing.T) {
	entryCount := 1000
	zone, err := NewSharedMetrics("TestNewSharedMetrics", metadataSize+entrySize*entryCount)
	if err != nil {
		t.Error(err)
	}
	defer zone.Free()

	for i := 0; i < entryCount; i++ {
		entry, err := zone.AllocEntry("testEntry" + strconv.Itoa(i))
		if err != nil {
			t.Error(err)
		}

		entry.value = int64(i + 1)
	}

	// re-alloc and re-access
	entry, err := zone.AllocEntry("testEntry0")
	if err != nil {
		t.Error(err)
	}

	if entry.value != 1 {
		t.Error("testEntry0 value not correct")
	}
}
