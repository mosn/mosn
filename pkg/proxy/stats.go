package proxy

import (
	"github.com/rcrowley/go-metrics"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stats"
)

const (
	DownstreamConnectionTotal   = "downstream_connection_total"
	DownstreamConnectionDestroy = "downstream_connection_destroy"
	DownstreamConnectionActive  = "downstream_connection_active"
	DownstreamBytesRead         = "downstream_bytes_read"
	DownstreamBytesReadCurrent  = "downstream_bytes_read_current"
	DownstreamBytesWrite        = "downstream_bytes_write"
	DownstreamBytesWriteCurrent = "downstream_bytes_write_current"
	DownstreamRequestTotal      = "downstream_request_total"
	DownstreamRequestActive     = "downstream_request_active"
	DownstreamRequestReset      = "downstream_request_reset"
	DownstreamRequestTime       = "downstream_request_time"
)

type proxyStats struct {
	stats *stats.Stats
}

func newProxyStats(namespace string) *proxyStats {
	return &proxyStats{
		stats: initProxyStats(namespace),
	}
}

func initProxyStats(namespace string) *stats.Stats {
	return stats.NewStats(namespace).AddCounter(DownstreamConnectionTotal).
		AddCounter(DownstreamConnectionDestroy).AddCounter(DownstreamConnectionActive).AddCounter(DownstreamBytesRead).
		AddGauge(DownstreamBytesReadCurrent).AddCounter(DownstreamBytesWrite).AddGauge(DownstreamBytesWriteCurrent).
		AddCounter(DownstreamRequestTotal).AddCounter(DownstreamRequestActive).AddCounter(DownstreamRequestReset).AddHistogram(DownstreamRequestTime)
}

func (s *proxyStats) DownstreamConnectionTotal() metrics.Counter {
	return s.stats.Counter(DownstreamConnectionTotal)
}

func (s *proxyStats) DownstreamConnectionDestroy() metrics.Counter {
	return s.stats.Counter(DownstreamConnectionDestroy)
}

func (s *proxyStats) DownstreamConnectionActive() metrics.Counter {
	return s.stats.Counter(DownstreamConnectionActive)
}

func (s *proxyStats) DownstreamBytesRead() metrics.Counter {
	return s.stats.Counter(DownstreamBytesRead)
}

func (s *proxyStats) DownstreamBytesReadCurrent() metrics.Gauge {
	return s.stats.Gauge(DownstreamBytesReadCurrent)
}

func (s *proxyStats) DownstreamBytesWrite() metrics.Counter {
	return s.stats.Counter(DownstreamBytesWrite)
}

func (s *proxyStats) DownstreamBytesWriteCurrent() metrics.Gauge {
	return s.stats.Gauge(DownstreamBytesWriteCurrent)
}

func (s *proxyStats) DownstreamRequestTotal() metrics.Counter {
	return s.stats.Counter(DownstreamRequestTotal)
}

func (s *proxyStats) DownstreamRequestActive() metrics.Counter {
	return s.stats.Counter(DownstreamRequestActive)
}

func (s *proxyStats) DownstreamRequestReset() metrics.Counter {
	return s.stats.Counter(DownstreamRequestReset)
}

func (s *proxyStats) DownstreamRequestTime() metrics.Histogram {
	return s.stats.Histogram(DownstreamRequestTime)
}

func (s *proxyStats) String() string {
	return s.stats.String()
}

type listenerStats struct {
	stats *stats.Stats
}

func newListenerStats(namespace string) *listenerStats {
	return &listenerStats{
		stats: initListenerStats(namespace),
	}
}

func initListenerStats(namespace string) *stats.Stats {
	return stats.NewStats(namespace).AddCounter(DownstreamRequestTotal).
		AddCounter(DownstreamRequestActive).AddCounter(DownstreamRequestReset).AddHistogram(DownstreamRequestTime)
}

func (s *listenerStats) DownstreamRequestTotal() metrics.Counter {
	return s.stats.Counter(DownstreamRequestTotal)
}

func (s *listenerStats) DownstreamRequestActive() metrics.Counter {
	return s.stats.Counter(DownstreamRequestActive)
}

func (s *listenerStats) DownstreamRequestReset() metrics.Counter {
	return s.stats.Counter(DownstreamRequestReset)
}

func (s *listenerStats) DownstreamRequestTime() metrics.Histogram {
	return s.stats.Histogram(DownstreamRequestTime)
}

func (s *listenerStats) String() string {
	return s.stats.String()
}
