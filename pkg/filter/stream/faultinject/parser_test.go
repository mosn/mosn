package faultinject

import (
	"testing"
	"time"
)

func TestParseStreamFaultInjectFilter(t *testing.T) {
	m := map[string]interface{}{
		"delay": map[string]interface{}{
			"fixed_delay": "1s",
			"percentage":  100,
		},
		"abort": map[string]interface{}{
			"status":     500,
			"percentage": 100,
		},
		"upstream_cluster": "clustername",
		"headers": []interface{}{
			map[string]interface{}{
				"name":  "service",
				"value": "test",
				"regex": false,
			},
			map[string]interface{}{
				"name":  "user",
				"value": "bob",
				"regex": false,
			},
		},
	}
	faultInject, err := ParseStreamFaultInjectFilter(m)
	if err != nil {
		t.Error("parse stream fault inject failed")
		return
	}
	if !(faultInject.UpstreamCluster == "clustername" &&
		len(faultInject.Headers) == 2 &&
		faultInject.Abort != nil &&
		faultInject.Delay != nil) {
		t.Error("parse stream fault inject unexpected")
		return
	}
	if !(faultInject.Abort.Percent == 100 && faultInject.Abort.Status == 500) {
		t.Error("parse stream fault inject's abort unexpected")
	}
	if !(faultInject.Delay.Percent == 100 && faultInject.Delay.Delay == time.Second) {
		t.Error("parse stream fault inject's delay unexpected")
	}
}
