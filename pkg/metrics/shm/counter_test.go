package shm

import (
	"testing"
	"unsafe"
)

func TestCounter(t *testing.T) {
	// 4096 * 100
	zone := InitMetricsZone("TestCounter", 4096*100)
	defer zone.Detach()

	entry, err := defaultZone.alloc("TestCounter")
	if err != nil {
		t.Fatal(err)
	}
	// inc
	counter := ShmCounter(unsafe.Pointer(&entry.value))
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
