package server

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/stats"
	"github.com/rcrowley/go-metrics"
)

const (
	// ~~~ stats names
	DownstreamConnectionTotal   = "downstream_connection_total"
	DownstreamConnectionDestroy = "downstream_connection_destroy"
	DownstreamConnectionActive  = "downstream_connection_active"
	DownstreamBytesRead         = "downstream_bytes_read"
	DownstreamBytesReadCurrent  = "downstream_bytes_read_current"
	DownstreamBytesWrite        = "downstream_bytes_write"
	DownstreamBytesWriteCurrent = "downstream_bytes_write_current"
)

type ListenerStats struct {
	stats *stats.Stats
}

func newListenerStats(namespace string) *ListenerStats {
	return &ListenerStats{
		stats: initStats(namespace),
	}
}

func initStats(namespace string) *stats.Stats {
	return stats.NewStats(namespace).AddCounter(DownstreamConnectionTotal).AddCounter(DownstreamConnectionDestroy).
		AddCounter(DownstreamConnectionActive).AddCounter(DownstreamBytesRead).
		AddGauge(DownstreamBytesReadCurrent).AddCounter(DownstreamBytesWrite).
		AddGauge(DownstreamBytesWriteCurrent)
}

func (ls *ListenerStats) DownstreamConnectionTotal() metrics.Counter {
	return ls.stats.Counter(DownstreamConnectionTotal)
}

func (ls *ListenerStats) DownstreamConnectionDestroy() metrics.Counter {
	return ls.stats.Counter(DownstreamConnectionDestroy)
}

func (ls *ListenerStats) DownstreamConnectionActive() metrics.Counter {
	return ls.stats.Counter(DownstreamConnectionActive)
}

func (ls *ListenerStats) DownstreamBytesRead() metrics.Counter {
	return ls.stats.Counter(DownstreamBytesRead)
}

func (ls *ListenerStats) DownstreamBytesReadCurrent() metrics.Gauge {
	return ls.stats.Gauge(DownstreamBytesReadCurrent)
}

func (ls *ListenerStats) DownstreamBytesWrite() metrics.Counter {
	return ls.stats.Counter(DownstreamBytesWrite)
}

func (ls *ListenerStats) DownstreamBytesWriteCurrent() metrics.Gauge {
	return ls.stats.Gauge(DownstreamBytesWriteCurrent)
}

func (ls *ListenerStats) String() string {
	return ls.stats.String()
}
