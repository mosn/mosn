package metric

import (
	"sort"
	"sync"
	"time"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/config"
	"github.com/alibaba/sentinel-golang/core/stat"
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
)

type metricTimeMap = map[uint64][]*base.MetricItem

const (
	logFlushQueueSize = 60
)

var (
	logger = logging.GetDefaultLogger()
	// The timestamp of the last fetching. The time unit is ms (= second * 1000).
	lastFetchTime int64 = -1
	writeChan           = make(chan metricTimeMap, logFlushQueueSize)
	stopChan            = make(chan struct{})

	metricWriter MetricLogWriter
	initOnce     sync.Once
)

func InitTask() (err error) {
	initOnce.Do(func() {
		metricWriter, err = NewDefaultMetricLogWriter(config.MetricLogSingleFileMaxSize(), config.MetricLogMaxFileAmount())
		if err != nil {
			logger.Errorf("Failed to initialize the MetricLogWriter: %+v", err)
			return
		}

		// Schedule the log flushing task
		go util.RunWithRecover(writeTaskLoop, logger)
		// Schedule the log aggregating task
		flushInterval := config.MetricLogFlushIntervalSec()
		if flushInterval == 0 {
			return
		}
		ticker := time.NewTicker(time.Duration(flushInterval) * time.Second)
		go util.RunWithRecover(func() {
			for {
				select {
				case <-ticker.C:
					doAggregate()
				case <-stopChan:
					ticker.Stop()
					return
				}
			}
		}, logger)
	})
	return err
}

// StopTask stops the task.
func StopTask() {
	stopChan <- struct{}{}
}

func writeTaskLoop() {
	for {
		select {
		case m := <-writeChan:
			keys := make([]uint64, 0, len(m))
			for t := range m {
				keys = append(keys, t)
			}
			// Sort the time
			sort.Slice(keys, func(i, j int) bool {
				return keys[i] < keys[j]
			})

			for _, t := range keys {
				err := metricWriter.Write(t, m[t])
				if err != nil {
					logger.Errorf("[MetricAggregatorTask] Write metric error: %+v", err)
				}
			}
		}
	}
}

func doAggregate() {
	curTime := util.CurrentTimeMillis()
	curTime = curTime - curTime%1000

	if int64(curTime) <= lastFetchTime {
		return
	}
	maps := make(metricTimeMap)
	cns := stat.ResourceNodeList()
	for _, node := range cns {
		metrics := currentMetricItems(node, curTime)
		aggregateIntoMap(maps, metrics, node)
	}
	// Aggregate for inbound entrance node.
	aggregateIntoMap(maps, currentMetricItems(stat.InboundNode(), curTime), stat.InboundNode())

	// Update current last fetch timestamp.
	lastFetchTime = int64(curTime)

	if len(maps) > 0 {
		writeChan <- maps
	}
}

func aggregateIntoMap(mm metricTimeMap, metrics map[uint64]*base.MetricItem, node *stat.ResourceNode) {
	for t, item := range metrics {
		item.Resource = node.ResourceName()
		item.Classification = int32(node.ResourceType())
		items, exists := mm[t]
		if exists {
			mm[t] = append(items, item)
		} else {
			mm[t] = []*base.MetricItem{item}
		}
	}
}

func isActiveMetricItem(item *base.MetricItem) bool {
	return item.PassQps > 0 || item.BlockQps > 0 || item.CompleteQps > 0 || item.ErrorQps > 0 ||
		item.AvgRt > 0 || item.Concurrency > 0
}

func isItemTimestampInTime(ts uint64, currentSecStart uint64) bool {
	// The bucket should satisfy: windowStart between [lastFetchTime, curStart)
	return int64(ts) >= lastFetchTime && ts < currentSecStart
}

func currentMetricItems(retriever base.MetricItemRetriever, currentTime uint64) map[uint64]*base.MetricItem {
	m := make(map[uint64]*base.MetricItem, 2)
	items := retriever.MetricsOnCondition(func(ts uint64) bool {
		return isItemTimestampInTime(ts, currentTime)
	})
	for _, item := range items {
		if !isActiveMetricItem(item) {
			continue
		}
		m[item.Timestamp] = item
	}
	return m
}
