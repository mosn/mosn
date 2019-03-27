package shm

import (
	"testing"
)

func TestCounter(t *testing.T) {
	// 4096 * 100
	zone, err := NewSharedMetrics("TestCounter", 4096*100)
	if err != nil {
		t.Fatal(err)
	}
	defer zone.Free()

	entry, err := zone.AllocEntry("TestCounter")
	if err != nil {
		t.Fatal(err)
	}
	// inc
	counter := NewShmCounter(entry)
	counter.Inc(5)

	if counter.Count() != 5 {
		t.Error("count ops failed")
	}

	// dec
	counter.Dec(2)
	if counter.Count() != 3 {
		t.Error("count ops failed")
	}

	// clear
	counter.Clear()
	if counter.Count() != 0 {
		t.Error("count ops failed")
	}
}
