package base

type TimePredicate func(uint64) bool

type MetricEvent int8

// There are five events to record
// pass + block == Total
const (
	// sentinel rules check pass
	MetricEventPass MetricEvent = iota
	// sentinel rules check block
	MetricEventBlock

	MetricEventComplete
	// Biz error, used for circuit breaker
	MetricEventError
	// request execute rt, unit is millisecond
	MetricEventRt
	// Specific for verifying mode
	MetricEventMonitorBlock
	// hack for the number of event
	MetricEventTotal
)

type ReadStat interface {
	GetQPS(event MetricEvent) float64
	GetSum(event MetricEvent) int64

	MinRT() float64
	AvgRT() float64
}

type WriteStat interface {
	AddMetric(event MetricEvent, count uint64)
}

// StatNode holds real-time statistics for resources.
type StatNode interface {
	MetricItemRetriever

	ReadStat
	WriteStat

	CurrentGoroutineNum() int32
	IncreaseGoroutineNum()
	DecreaseGoroutineNum()

	Reset()
}
