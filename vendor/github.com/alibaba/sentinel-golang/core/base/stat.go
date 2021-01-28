// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package base

import (
	"errors"
)

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

var (
	globalNopReadStat  = &nopReadStat{}
	globalNopWriteStat = &nopWriteStat{}
)

type ReadStat interface {
	GetQPS(event MetricEvent) float64
	GetPreviousQPS(event MetricEvent) float64
	GetSum(event MetricEvent) int64

	MinRT() float64
	AvgRT() float64
}

func NopReadStat() *nopReadStat {
	return globalNopReadStat
}

type nopReadStat struct {
}

func (rs *nopReadStat) GetQPS(_ MetricEvent) float64 {
	return 0.0
}

func (rs *nopReadStat) GetPreviousQPS(_ MetricEvent) float64 {
	return 0.0
}

func (rs *nopReadStat) GetSum(_ MetricEvent) int64 {
	return 0
}

func (rs *nopReadStat) MinRT() float64 {
	return 0.0
}

func (rs *nopReadStat) AvgRT() float64 {
	return 0.0
}

type WriteStat interface {
	AddCount(event MetricEvent, count int64)
}

func NopWriteStat() *nopWriteStat {
	return globalNopWriteStat
}

type nopWriteStat struct {
}

func (ws *nopWriteStat) AddCount(_ MetricEvent, _ int64) {
}

type ConcurrencyStat interface {
	CurrentConcurrency() int32
	IncreaseConcurrency()
	DecreaseConcurrency()
}

// StatNode holds real-time statistics for resources.
type StatNode interface {
	MetricItemRetriever

	ReadStat
	WriteStat
	ConcurrencyStat

	// GenerateReadStat generates the readonly metric statistic based on resource level global statistic
	// If parameters, sampleCount and intervalInMs, are not suitable for resource level global statistic, return (nil, error)
	GenerateReadStat(sampleCount uint32, intervalInMs uint32) (ReadStat, error)
}

var (
	IllegalGlobalStatisticParamsError = errors.New("Invalid parameters, sampleCount or interval, for resource's global statistic")
	IllegalStatisticParamsError       = errors.New("Invalid parameters, sampleCount or interval, for metric statistic")
	GlobalStatisticNonReusableError   = errors.New("The parameters, sampleCount and interval, mismatch for reusing between resource's global statistic and readonly metric statistic.")
)

func CheckValidityForStatistic(sampleCount, intervalInMs uint32) error {
	if intervalInMs == 0 || sampleCount == 0 || intervalInMs%sampleCount != 0 {
		return IllegalStatisticParamsError
	}
	return nil
}

// CheckValidityForReuseStatistic check the compliance whether readonly metric statistic can be built based on resource's global statistic
// The parameters, sampleCount and intervalInMs, are the parameters of the metric statistic you want to build
// The parameters, parentSampleCount and parentIntervalInMs, are the parameters of the resource's global statistic
// If compliance passes, return nil, if not returns specific error
func CheckValidityForReuseStatistic(sampleCount, intervalInMs uint32, parentSampleCount, parentIntervalInMs uint32) error {
	if intervalInMs == 0 || sampleCount == 0 || intervalInMs%sampleCount != 0 {
		return IllegalStatisticParamsError
	}
	bucketLengthInMs := intervalInMs / sampleCount

	if parentIntervalInMs == 0 || parentSampleCount == 0 || parentIntervalInMs%parentSampleCount != 0 {
		return IllegalGlobalStatisticParamsError
	}
	parentBucketLengthInMs := parentIntervalInMs / parentSampleCount

	//SlidingWindowMetric's intervalInMs is not divisible by BucketLeapArray's intervalInMs
	if parentIntervalInMs%intervalInMs != 0 {
		return GlobalStatisticNonReusableError
	}
	// BucketLeapArray's BucketLengthInMs is not divisible by SlidingWindowMetric's BucketLengthInMs
	if bucketLengthInMs%parentBucketLengthInMs != 0 {
		return GlobalStatisticNonReusableError
	}
	return nil
}
