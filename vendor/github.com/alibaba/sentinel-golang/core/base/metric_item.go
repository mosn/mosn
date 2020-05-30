package base

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/alibaba/sentinel-golang/util"
)

const metricPartSeparator = "|"

// MetricItem represents the data of metric log per line.
type MetricItem struct {
	Resource             string
	Classification       int32
	Timestamp            uint64
	PassQps              uint64
	BlockQps             uint64
	MonitorBlockQps      uint64
	CompleteQps          uint64
	ErrorQps             uint64
	AvgRt                uint64
	OccupiedPassQps      uint64
	Concurrency          uint32
	SecondMaxConcurrency uint32
}

type MetricItemRetriever interface {
	MetricsOnCondition(predicate TimePredicate) []*MetricItem
}

func (m *MetricItem) ToFatString() (string, error) {
	b := strings.Builder{}
	timeStr := util.FormatTimeMillis(m.Timestamp)
	// All "|" in the resource name will be replaced with "_"
	finalName := strings.ReplaceAll(m.Resource, "|", "_")
	_, err := fmt.Fprintf(&b, "%d|%s|%s|%d|%d|%d|%d|%d|%d|%d|%d|%d",
		m.Timestamp, timeStr, finalName, m.PassQps,
		m.BlockQps, m.CompleteQps, m.ErrorQps, m.AvgRt,
		m.OccupiedPassQps, m.Concurrency, m.Classification, m.MonitorBlockQps)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func (m *MetricItem) ToThinString() (string, error) {
	b := strings.Builder{}
	finalName := strings.ReplaceAll(m.Resource, "|", "_")
	_, err := fmt.Fprintf(&b, "%d|%s|%d|%d|%d|%d|%d|%d|%d|%d|%d",
		m.Timestamp, finalName, m.PassQps,
		m.BlockQps, m.CompleteQps, m.ErrorQps, m.AvgRt,
		m.OccupiedPassQps, m.Concurrency, m.Classification, m.MonitorBlockQps)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func MetricItemFromFatString(line string) (*MetricItem, error) {
	if len(line) == 0 {
		return nil, errors.New("invalid metric line: empty string")
	}
	item := &MetricItem{}
	arr := strings.Split(line, metricPartSeparator)
	if len(arr) < 8 {
		return nil, errors.New("invalid metric line: invalid format")
	}
	ts, err := strconv.ParseUint(arr[0], 10, 64)
	if err != nil {
		return nil, err
	}
	item.Timestamp = ts
	item.Resource = arr[2]
	p, err := strconv.ParseUint(arr[3], 10, 64)
	if err != nil {
		return nil, err
	}
	item.PassQps = p
	b, err := strconv.ParseUint(arr[4], 10, 64)
	if err != nil {
		return nil, err
	}
	item.BlockQps = b
	c, err := strconv.ParseUint(arr[5], 10, 64)
	if err != nil {
		return nil, err
	}
	item.CompleteQps = c
	e, err := strconv.ParseUint(arr[6], 10, 64)
	if err != nil {
		return nil, err
	}
	item.ErrorQps = e
	rt, err := strconv.ParseUint(arr[7], 10, 64)
	if err != nil {
		return nil, err
	}
	item.AvgRt = rt

	if len(arr) >= 9 {
		oc, err := strconv.ParseUint(arr[8], 10, 64)
		if err != nil {
			return nil, err
		}
		item.OccupiedPassQps = oc
	}
	if len(arr) >= 10 {
		concurrency, err := strconv.ParseUint(arr[9], 10, 32)
		if err != nil {
			return nil, err
		}
		item.Concurrency = uint32(concurrency)
	}
	if len(arr) >= 11 {
		cl, err := strconv.ParseInt(arr[10], 10, 32)
		if err != nil {
			return nil, err
		}
		item.Classification = int32(cl)
	}
	if len(arr) >= 12 {
		mb, err := strconv.ParseUint(arr[11], 10, 64)
		if err != nil {
			return nil, err
		}
		item.MonitorBlockQps = mb
	}
	return item, nil
}
