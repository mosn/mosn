package shm

import (
	"testing"
)

func TestGauge(t *testing.T) {
	// 4096 * 100
	zone, _ := NewSharedMetrics("TestGauge", 4096*100)
	defer zone.Free()

	entry, _ := zone.AllocEntry("TestGauge")

	gauge := NewShmGauge(entry)

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
