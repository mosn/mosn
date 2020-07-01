/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package regulator

import (
	"sync/atomic"
	"testing"
	"time"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

func Test_IsArrivalTime(t *testing.T) {
	//
	config := &v2.FaultToleranceFilterConfig{
		Enabled:               true,
		ExceptionTypes:        map[uint32]bool{},
		TimeWindow:            3000,
		LeastWindowCount:      10,
		ExceptionRateMultiple: 5,
		MaxIpCount:            2,
		MaxIpRatio:            0.2,
		RecoverTime:           1000,
	}
	//
	model := NewMeasureModel("aaa", config)
	if model.IsArrivalTime() {
		t.Error("Test_IsArrivalTime Failed")
	}
	if model.IsArrivalTime() {
		t.Error("Test_IsArrivalTime Failed")
	}
	//
	time.Sleep(3 * time.Second)
	if !model.IsArrivalTime() {
		t.Error("Test_IsArrivalTime Failed")
	}

	//
	time.Sleep(3 * time.Second)
	if !model.IsArrivalTime() {
		t.Error("Test_IsArrivalTime Failed")
	}
}

func Test_Measure(t *testing.T) {
	//
	initInvocationStatFactory()
	//
	config := &v2.FaultToleranceFilterConfig{
		Enabled:               true,
		ExceptionTypes:        map[uint32]bool{},
		TimeWindow:            3000,
		LeastWindowCount:      10,
		ExceptionRateMultiple: 2,
		MaxIpCount:            2,
		RecoverTime:           1000,
	}
	//
	stat_1 := buildTestInvocationStat("1")
	stat_2 := buildTestInvocationStat("2")
	stat_3 := buildTestInvocationStat("3")
	stat_4 := buildTestInvocationStat("4")
	stat_5 := buildTestInvocationStat("5")
	model := NewMeasureModel("measureKey", config)
	model.AddInvocationStat(stat_1)
	model.AddInvocationStat(stat_2)
	model.AddInvocationStat(stat_3)
	model.AddInvocationStat(stat_4)
	model.AddInvocationStat(stat_5)
	//
	model.Measure()
	assertMeasureModelCount(t, model, 5, 0)
	assertStatHealthy(t, stat_1, true)
	assertStatHealthy(t, stat_2, true)
	assertStatHealthy(t, stat_3, true)
	assertStatHealthy(t, stat_4, true)
	assertStatHealthy(t, stat_5, true)
	//
	callInvocationStat(stat_3, 10, 10)
	model.Measure()
	assertMeasureModelCount(t, model, 5, 0)
	assertStatHealthy(t, stat_1, true)
	//0.25 0.4
	callInvocationStat(stat_1, 10, 1)
	callInvocationStat(stat_2, 10, 4)
	model.Measure()
	assertMeasureModelCount(t, model, 5, 0)
	assertStatHealthy(t, stat_1, true)
	assertStatHealthy(t, stat_2, true)
	//
	config.ExceptionRateMultiple = 1.6
	callInvocationStat(stat_1, 10, 1)
	callInvocationStat(stat_2, 10, 4)
	model.Measure()
	assertMeasureModelCount(t, model, 5, 1)
	assertStatHealthy(t, stat_1, true)
	assertStatHealthy(t, stat_2, false)
	//
	model.Measure()
	assertMeasureModelCount(t, model, 5, 1)
	model.Measure()
	assertMeasureModelCount(t, model, 3, 1)
	assertStatExist(t, model, stat_3, true)
	assertStatExist(t, model, stat_4, false)
	assertStatExist(t, model, stat_5, false)
}

func assertStatExist(t *testing.T, model *MeasureModel, stat *InvocationStat, isExist bool) {
	if _, ok := model.stats.Load(stat.GetInvocationKey()); ok == isExist {
		t.Log("assertStatExist Success")
	} else {
		t.Error("assertStatExist Failed")
	}
	if _, ok := GetInvocationStatFactoryInstance().invocationStats.Load(stat.GetInvocationKey()); ok == isExist {
		t.Log("assertStatExist Success")
	} else {
		t.Error("assertStatExist Failed")
	}
}

func callInvocationStat(stat *InvocationStat, totalCall uint64, exceptionCall uint64) {
	healthyCount := totalCall - exceptionCall
	var i uint64
	for i = 0; i < healthyCount; i++ {
		stat.Call(false)
	}
	var j uint64
	for j = 0; j < exceptionCall; j++ {
		stat.Call(true)
	}
}

func assertStatHealthy(t *testing.T, stat *InvocationStat, isHealthy bool) {
	if stat.IsHealthy() == isHealthy {
		t.Log("assertStatHealthy Success")
	} else {
		t.Error("assertStatHealthy Failed")
	}
}

func assertMeasureModelCount(t *testing.T, model *MeasureModel, count uint64, downGradeCount uint64) {
	if model.count == count {
		t.Log("assertMeasureModelCount Success")
	} else {
		t.Error("assertMeasureModelCount Failed")
	}
	if model.downgradeCount == downGradeCount {
		t.Log("assertMeasureModelCount Success")
	} else {
		t.Error("assertMeasureModelCount Failed")
	}
}

func buildTestInvocationStat(index string) *InvocationStat {
	dimension := NewMockInvocationDimension("invocationKey_"+index, "measureKey")
	host := NewMockHostInfo()
	stat := GetInvocationStatFactoryInstance().GetInvocationStat(host, dimension)
	return stat
}

func initInvocationStatFactory() {
	config := &v2.FaultToleranceFilterConfig{
		TaskSize: 10,
	}
	invocationStatFactory := NewInvocationStatFactory(config)
	invocationStatFactoryInstance = invocationStatFactory
}

type MockHostInfo struct {
	healthy *int32
}

func NewMockHostInfo() api.HostInfo {
	var value int32 = 0
	return MockHostInfo{
		healthy: &value,
	}
}

func (m MockHostInfo) Hostname() string {
	panic("implement me")
}

func (m MockHostInfo) Metadata() api.Metadata {
	panic("implement me")
}

func (m MockHostInfo) AddressString() string {
	panic("implement me")
}

func (m MockHostInfo) Weight() uint32 {
	panic("implement me")
}

func (m MockHostInfo) SupportTLS() bool {
	panic("implement me")
}

func (m MockHostInfo) ClearHealthFlag(flag api.HealthFlag) {
	atomic.StoreInt32(m.healthy, 0)
}

func (m MockHostInfo) ContainHealthFlag(flag api.HealthFlag) bool {
	panic("implement me")
}

func (m MockHostInfo) SetHealthFlag(flag api.HealthFlag) {
	atomic.StoreInt32(m.healthy, 1)
}

func (m MockHostInfo) HealthFlag() api.HealthFlag {
	panic("implement me")
}

func (m MockHostInfo) Health() bool {
	return (*m.healthy) == 0
}
