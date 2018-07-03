package stats

import (
	"bytes"
	"fmt"
	"github.com/rcrowley/go-metrics"
	"strconv"
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

	//buffer.WriteString(fmt.Sprintf("namespace: %s, ", s.namespace))
	buffer.WriteString("namespace: " + s.namespace + ", ")

	if len(s.counters) > 0 {
		buffer.WriteString("counters: [")

		for name, counter := range s.counters {
			buffer.WriteString(name + ": " + strconv.FormatInt(counter.Count(), 10))
		}

		buffer.WriteString("], ")
	}

	if len(s.gauges) > 0 {
		buffer.WriteString("gauges: [")

		for name, gauge := range s.gauges {
			buffer.WriteString(name + ": " + strconv.FormatInt(gauge.Value(), 10))
		}

		buffer.WriteString("], ")
	}

	if len(s.histograms) > 0 {
		buffer.WriteString("histograms: [")

		for name, histogram := range s.histograms {
			buffer.WriteString(name + ": " + strconv.FormatInt(histogram.Count(), 10))
		}

		buffer.WriteString("]")
	}

	return buffer.String()
}
