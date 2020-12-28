package metrics

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDownstreamMetrics(t *testing.T) {
	proxyMetrics := NewProxyStats("test_proxy")

	counter := proxyMetrics.Counter("request_count")
	counter.Inc(100)
	assert.Equal(t, counter.Count(), int64(100))
}

func TestHealthStats(t *testing.T) {
	healthStats := NewHealthStats("test_service")
	counter := healthStats.Counter("upup")
	counter.Inc(999)
	assert.Equal(t, counter.Count(), int64(999))
}

func TestMosnMetrics(t *testing.T) {
	FlushMosnMetrics = true
	defer func() {
		FlushMosnMetrics = false
	}()

	m := NewMosnMetrics()
	c := m.Counter("mosn_mosn")
	c.Inc(100)
	c.Dec(10)
	assert.Equal(t, c.Count(), int64(90))
}

func TestUseless(t *testing.T) {
	// useless functions test
	SetGoVersion("go1.19")
	SetVersion("mosn1.13")
	SetStateCode(1)
	AddListenerAddr("localhost:1111")
}
