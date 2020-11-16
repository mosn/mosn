package metrics

import (
	"errors"
	"sync"

	gometrics "github.com/rcrowley/go-metrics"
)

type lazyCounter struct {
	once    sync.Once
	ctor    func() gometrics.Counter
	counter gometrics.Counter
}

// NewLazyCounter build lazyCounter
func NewLazyCounter(ctor func() gometrics.Counter) (gometrics.Counter, error) {
	if ctor == nil {
		return nil, errors.New("build lazycounter ctor is empty")
	}
	return &lazyCounter{ctor: ctor}, nil
}

func (lc *lazyCounter) preFunc() {
	lc.once.Do(func() {
		lc.counter = lc.ctor()
	})
}

func (lc *lazyCounter) Clear() {
	lc.preFunc()
	lc.counter.Clear()
}

func (lc *lazyCounter) Count() int64 {
	lc.preFunc()
	return lc.counter.Count()
}

func (lc *lazyCounter) Dec(value int64) {
	lc.preFunc()
	lc.counter.Dec(value)
}

func (lc *lazyCounter) Inc(value int64) {
	lc.preFunc()
	lc.counter.Inc(value)
}

func (lc *lazyCounter) Snapshot() gometrics.Counter {
	lc.preFunc()
	return lc.counter.Snapshot()
}

type lazyGauge struct {
	once  sync.Once
	ctor  func() gometrics.Gauge
	gauge gometrics.Gauge
}

// NewLazyGauge build lazyGauge
func NewLazyGauge(ctor func() gometrics.Gauge) (gometrics.Gauge, error) {
	if ctor == nil {
		return nil, errors.New("build lazygauge ctor is empty")
	}
	return &lazyGauge{ctor: ctor}, nil
}

func (lg *lazyGauge) preFunc() {
	lg.once.Do(func() {
		lg.gauge = lg.ctor()
	})
}

func (lg *lazyGauge) Update(value int64) {
	lg.preFunc()
	lg.gauge.Update(value)
}

func (lg *lazyGauge) Value() int64 {
	lg.preFunc()
	return lg.gauge.Value()
}

func (lg *lazyGauge) Snapshot() gometrics.Gauge {
	lg.preFunc()
	return lg.gauge.Snapshot()
}

type lazyHistogram struct {
	once      sync.Once
	ctor      func() gometrics.Histogram
	histogram gometrics.Histogram
}

// NewLazyHistogram build lazyHistogram
func NewLazyHistogram(ctor func() gometrics.Histogram) (gometrics.Histogram, error) {
	if ctor == nil {
		return nil, errors.New("build lazygauge ctor is empty")
	}
	return &lazyHistogram{ctor: ctor}, nil
}

func (lh *lazyHistogram) preFunc() {
	lh.once.Do(func() {
		lh.histogram = lh.ctor()
	})
}

func (lh *lazyHistogram) Clear() {
	lh.preFunc()
	lh.histogram.Clear()
}

func (lh *lazyHistogram) Count() int64 {
	lh.preFunc()
	return lh.histogram.Count()
}

func (lh *lazyHistogram) Max() int64 {
	lh.preFunc()
	return lh.histogram.Max()
}

func (lh *lazyHistogram) Mean() float64 {
	lh.preFunc()
	return lh.histogram.Mean()
}

func (lh *lazyHistogram) Min() int64 {
	lh.preFunc()
	return lh.histogram.Min()
}

func (lh *lazyHistogram) Percentile(f float64) float64 {
	lh.preFunc()
	return lh.histogram.Percentile(f)
}

func (lh *lazyHistogram) Percentiles(f []float64) []float64 {
	lh.preFunc()
	return lh.histogram.Percentiles(f)
}

func (lh *lazyHistogram) Sample() gometrics.Sample {
	lh.preFunc()
	return lh.histogram.Sample()
}

func (lh *lazyHistogram) Snapshot() gometrics.Histogram {
	lh.preFunc()
	return lh.histogram.Snapshot()
}

func (lh *lazyHistogram) StdDev() float64 {
	lh.preFunc()
	return lh.histogram.StdDev()
}

func (lh *lazyHistogram) Sum() int64 {
	lh.preFunc()
	return lh.histogram.Sum()
}

func (lh *lazyHistogram) Update(value int64) {
	lh.preFunc()
	lh.histogram.Update(value)
}

func (lh *lazyHistogram) Variance() float64 {
	lh.preFunc()
	return lh.histogram.Variance()
}
