package metrics

import (
	"testing"

	gometrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"

	"mosn.io/api"
)

func TestNilMetricsIsMetrics(t *testing.T) {
	m, err := NewNilMetrics(MosnMetaType, nil)
	assert.NoError(t, err)
	_, ok := m.(api.Metrics)
	assert.True(t, ok)
}

func TestNilMetricsSuppliers(t *testing.T) {
	m, _ := NewNilMetrics(MosnMetaType, nil)
	c := m.Counter("counter")
	assert.IsType(t, gometrics.NilCounter{}, c)

	g := m.Gauge("gauge")
	assert.IsType(t, gometrics.NilGauge{}, g)

	h := m.Histogram("histogram")
	assert.IsType(t, gometrics.NilHistogram{}, h)

	e := m.EWMA("ewma", 0.1)
	assert.IsType(t, gometrics.NilEWMA{}, e)
}
