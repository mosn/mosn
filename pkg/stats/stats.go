package stats

import (
	"fmt"
	"bytes"
	"github.com/rcrowley/go-metrics"
)

type Stats struct {
	namespace  string
	counters   map[string]metrics.Counter
	gauges     map[string]metrics.Gauge
	histograms map[string]metrics.Histogram
}

func NewStats(namespace string) *Stats {
	return &Stats{
		namespace:  namespace,
		counters:   make(map[string]metrics.Counter),
		gauges:     make(map[string]metrics.Gauge),
		histograms: make(map[string]metrics.Histogram),
	}
}

func (s *Stats) AddCounter(name string) *Stats {
	metricsKey := fmt.Sprintf("%s.%s", s.namespace, name)
	s.counters[name] = metrics.GetOrRegisterCounter(metricsKey, nil)

	return s
}

func (s *Stats) AddGauge(name string) *Stats {
	metricsKey := fmt.Sprintf("%s.%s", s.namespace, name)
	s.gauges[name] = metrics.GetOrRegisterGauge(metricsKey, nil)

	return s
}

func (s *Stats) AddHistogram(name string) *Stats {
	metricsKey := fmt.Sprintf("%s.%s", s.namespace, name)
	s.histograms[name] = metrics.GetOrRegisterHistogram(metricsKey, nil, metrics.NewUniformSample(100))

	return s
}

func (s *Stats) SetCounter(name string, counter metrics.Counter) {
	s.counters[name] = counter
}

func (s *Stats) SetGauge(name string, gauge metrics.Gauge) {
	s.gauges[name] = gauge
}

func (s *Stats) SetHistogram(name string, histogram metrics.Histogram) {
	s.histograms[name] = histogram
}

func (s *Stats) Counter(name string) metrics.Counter {
	return s.counters[name]
}

func (s *Stats) Gauge(name string) metrics.Gauge {
	return s.gauges[name]
}

func (s *Stats) Histogram(name string) metrics.Histogram {
	return s.histograms[name]
}

func (s *Stats) String() string {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("namespace: %s, ", s.namespace))

	if len(s.counters) > 0 {
		buffer.WriteString(fmt.Sprintf("counters: ["))

		for name, counter := range s.counters {
			buffer.WriteString(fmt.Sprintf("%s: %d, ", name, counter.Count()))
		}

		buffer.WriteString(fmt.Sprintf("], "))
	}

	if len(s.gauges) > 0 {
		buffer.WriteString(fmt.Sprintf("gauges: ["))

		for name, gauge := range s.gauges {
			buffer.WriteString(fmt.Sprintf("%s: %d, ", name, gauge.Value()))
		}

		buffer.WriteString(fmt.Sprintf("], "))
	}

	if len(s.histograms) > 0 {
		buffer.WriteString(fmt.Sprintf("histograms: ["))

		for name, histogram := range s.histograms {
			buffer.WriteString(fmt.Sprintf("%s, %d ", name, histogram.Count()))
		}

		buffer.WriteString(fmt.Sprintf("]"))
	}

	return buffer.String()
}
