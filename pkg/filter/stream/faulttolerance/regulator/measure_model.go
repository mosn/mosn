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
	"fmt"
	"sync"
	"sync/atomic"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/filter/stream/faulttolerance/util"
)

type MeasureModel struct {
	key            string
	stats          *sync.Map
	count          uint64
	downgradeCount uint64
	timeMeter      int64
	config         *v2.FaultToleranceFilterConfig
}

func NewMeasureModel(key string, config *v2.FaultToleranceFilterConfig) *MeasureModel {
	measureModel := &MeasureModel{
		key:            key,
		stats:          new(sync.Map),
		count:          0,
		downgradeCount: 0,
		timeMeter:      0,
		config:         config,
	}
	return measureModel
}

func (m *MeasureModel) GetKey() string {
	return m.key
}

func (m *MeasureModel) AddInvocationStat(stat *InvocationStat) {
	key := stat.GetInvocationKey()
	if _, ok := m.stats.LoadOrStore(key, stat); !ok {
		atomic.AddUint64(&m.count, 1)
	}
}

func (m *MeasureModel) releaseInvocationStat(stat *InvocationStat) {
	key := stat.GetInvocationKey()
	m.stats.Delete(key)
	// To subtract a signed positive constant value c from x, do AddUint64(&x, ^uint64(c-1)).
	atomic.AddUint64(&m.count, ^uint64(0))
	GetInvocationStatFactoryInstance().ReleaseInvocationStat(key)
}

func (m *MeasureModel) Measure() {
	snapshots := m.snapshotInvocations(m.config.RecoverTime)
	ok, averageExceptionRate := m.calculateAverageExceptionRate(snapshots, m.config.LeastWindowCount)
	if !ok {
		return
	}
	for _, snapshot := range snapshots {
		call, _ := snapshot.GetCount()
		if call >= uint64(m.config.LeastWindowCount) {
			_, exceptionRate := snapshot.GetExceptionRate()
			multiple := util.DivideFloat64(exceptionRate, averageExceptionRate)
			if multiple >= m.config.ExceptionRateMultiple {
				m.downgrade(snapshot, m.config.MaxIpCount)
			}
		}
	}

	m.updateInvocationSnapshots(snapshots)
}

func (m *MeasureModel) downgrade(snapshot *InvocationStat, maxIpCount uint64) {
	if atomic.LoadUint64(&m.count) <= 0 {
		return
	}
	if m.downgradeCount+1 > maxIpCount {
		return
	}
	key := snapshot.GetInvocationKey()
	if value, ok := m.stats.Load(key); ok {
		stat := value.(*InvocationStat)
		stat.Downgrade()
		m.downgradeCount++
	}
}

func (m *MeasureModel) snapshotInvocations(recoverTime int64) []*InvocationStat {
	snapshots := []*InvocationStat{}
	m.stats.Range(func(app, value interface{}) bool {
		stat := value.(*InvocationStat)
		if !stat.IsHealthy() {
			m.recover(stat, recoverTime)
			return true
		}
		if stat.GetCall() <= 0 {
			// don't release unhealth stat host
			if stat.AddUselessCycle() && stat.IsHealthy() {
				m.releaseInvocationStat(stat)
			}
			return true
		} else {
			stat.RestUselessCycle()
		}
		snapshot := value.(*InvocationStat).Snapshot()
		snapshots = append(snapshots, snapshot)
		return true
	})
	return snapshots
}

func (m *MeasureModel) recover(stat *InvocationStat, recoverTime int64) {
	if downgradeTime := stat.GetDowngradeTime(); downgradeTime != 0 {
		now := util.GetNowMS()
		if now-downgradeTime >= recoverTime {
			stat.Recover()
			m.downgradeCount--
		}
	}
}

func (m *MeasureModel) updateInvocationSnapshots(snapshots []*InvocationStat) {
	for _, snapshot := range snapshots {
		if value, ok := m.stats.Load(snapshot.GetInvocationKey()); ok {
			stat := value.(*InvocationStat)
			stat.Update(snapshot)
		}
	}
}

func (m *MeasureModel) calculateAverageExceptionRate(stats []*InvocationStat, leastWindowCount int64) (bool, float64) {
	var sumException uint64
	var sumCall uint64
	for _, stat := range stats {
		if call, exception := stat.GetCount(); call >= uint64(leastWindowCount) {
			sumException += exception
			sumCall += call
		}
	}
	if sumCall == 0 {
		return false, 0
	}
	return true, util.DivideInt64(int64(sumException), int64(sumCall))
}

func (m *MeasureModel) IsArrivalTime() bool {
	timeWindow := m.config.TimeWindow
	now := util.GetNowMS()

	if m.timeMeter == 0 {
		m.timeMeter = now + timeWindow
		return false
	} else {
		if now >= m.timeMeter {
			m.timeMeter = now + timeWindow
			return true
		} else {
			return false
		}
	}
}

func (m *MeasureModel) String() string {
	str := fmt.Sprintf("key=%s,count=%v,downgradeCount=%v,timeMeter=%v",
		m.key, atomic.LoadUint64(&m.count), m.downgradeCount, m.timeMeter)
	return str
}
