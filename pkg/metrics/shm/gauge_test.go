package shm

import (
	"testing"
	"unsafe"
)

func TestGauge(t *testing.T) {
	// 4096 * 100
	zone := InitMetricsZone("TestGauge", 4096*100)
	defer zone.Detach()

	entry, err := defaultZone.alloc("TestGauge")
	if err != nil {
		t.Error(err)
	}

	gauge := ShmGauge(unsafe.Pointer(&entry.value))

	// update
	gauge.Update(5)

	// value
	if gauge.Value() != 5 {
		t.Error("gauge ops failed")
	}

	gauge.Update(123)
	if gauge.Value() != 123 {
		t.Error("gauge ops failed")
	}
}
