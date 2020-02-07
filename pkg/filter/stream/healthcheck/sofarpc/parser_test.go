package sofarpc

import (
	"testing"
	"time"
)

func TestParseHealthCheckFilter(t *testing.T) {
	m := map[string]interface{}{
		"passthrough": true,
		"cache_time":  "10m",
		"endpoint":    "test",
		"cluster_min_healthy_percentages": map[string]float64{
			"test": 10.0,
		},
	}
	healthCheck, _ := ParseHealthCheckFilter(m)
	if !(healthCheck.PassThrough &&
		healthCheck.CacheTime == 10*time.Minute &&
		healthCheck.Endpoint == "test" &&
		len(healthCheck.ClusterMinHealthyPercentage) == 1 &&
		healthCheck.ClusterMinHealthyPercentage["test"] == 10.0) {
		t.Error("parse health check filter failed")
	}
}
