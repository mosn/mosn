package faultinject

import (
	"testing"
	"time"
)

func TestParseFaultInjectFilter(t *testing.T) {
	m := map[string]interface{}{
		"delay_percent":  100,
		"delay_duration": "15s",
	}
	faultInject, _ := ParseFaultInjectFilter(m)
	if !(faultInject.DelayDuration == uint64(15*time.Second) && faultInject.DelayPercent == 100) {
		t.Error("parse fault inject failed")
	}
}
