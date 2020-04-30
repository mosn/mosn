package regulator

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type MeasureModel struct {
	key            string
	stats          *sync.Map
	count          int64
	downgradeCount int64
	timeMeter      int64
}

func NewMeasureModel(key string) *MeasureModel {
	measureModel := &MeasureModel{
		key:            key,
		stats:          new(sync.Map),
		count:          0,
		downgradeCount: 0,
		timeMeter:      0,
	}
	return measureModel
}

func (m *MeasureModel) GetKey() string {
	return m.key
}

func (m *MeasureModel) AddInvocationStat(stat *InvocationStat) {
	key := stat.GetInvocationKey()
	if _, ok := m.stats.LoadOrStore(key, stat); !ok {
		atomic.AddInt64(&m.count, 1)
	}
}

func (m *MeasureModel) releaseInvocationStat(stat *InvocationStat) {
	key := stat.GetInvocationKey()
	m.stats.Delete(key)
	atomic.AddInt64(&m.count, -1)
	GetInvocationStatFactoryInstance().ReleaseInvocationStat(key)
}

func (m *MeasureModel) Measure(rule *fault_tolerance_rule.FaultToleranceRule) {
	if tolerance_log.FaultToleranceLog.GetLogLevel() >= log.DEBUG {
		tolerance_log.FaultToleranceLog.Debugf("[Tolerance][MeasureModel][%s] measure before, measureModel = %v", m.key, m)
	}
	snapshots := m.snapshotInvocations()
	if tolerance_log.FaultToleranceLog.GetLogLevel() >= log.DEBUG {
		tolerance_log.FaultToleranceLog.Debugf("[Tolerance][MeasureModel][%s] get snapshots, measureModel = %v, snapshot = %v", m.key, m, snapshots)
	}
	ok, averageExceptionRate := m.calculateAverageExceptionRate(snapshots, rule.GetLeastWindowCount())
	if tolerance_log.FaultToleranceLog.GetLogLevel() >= log.DEBUG {
		tolerance_log.FaultToleranceLog.Debugf("[Tolerance][MeasureModel][%s] get averageExceptionRate, ok = %v, averageExceptionRate = %v", m.key, ok, averageExceptionRate)
	}
	if !ok {
		return
	}
	for _, snapshot := range snapshots {
		call, _ := snapshot.GetCount()
		if call >= rule.GetLeastWindowCount() {
			_, exceptionRate := snapshot.GetExceptionRate()
			multiple := common.DivideFloat64(exceptionRate, averageExceptionRate)
			if multiple >= rule.GetExceptionRateMultiple() {
				if m.downgrade(snapshot, rule.GetMaxIpCount(), rule.GetMaxIpRatio()) {
					m.report(snapshot)
					if tolerance_log.FaultToleranceLog.GetLogLevel() >= log.DEBUG {
						tolerance_log.FaultToleranceLog.Debugf("[Tolerance][MeasureModel][%s] downgrade a snapshot, report, snapshot= %v", m.key, snapshot)
					}
				} else {
					if tolerance_log.FaultToleranceLog.GetLogLevel() >= log.DEBUG {
						tolerance_log.FaultToleranceLog.Debugf("[Tolerance][MeasureModel][%s] downgrade failed, snapshot= %v", m.key, snapshot)
					}
				}
			}
		}
	}

	m.updateInvocationSnapshots(snapshots)
	if tolerance_log.FaultToleranceLog.GetLogLevel() >= log.DEBUG {
		tolerance_log.FaultToleranceLog.Debugf("[Tolerance][MeasureModel][%s] measure after", m.key)
	}
}

func (m *MeasureModel) report(stat *InvocationStat) {
	metrics := report.GetFaultToleranceMetricsManagerInstance().GetMetrics(stat.GetMetricsKey(), stat.GetAppName(), stat.GetAddress())
	metrics.Report(stat.callCount, stat.exceptionCount)
}

func (m *MeasureModel) downgrade(snapshot *InvocationStat, maxIpCount int64, maxIpRatio float64) bool {
	if m.count <= 0 {
		return false
	}
	if m.downgradeCount+1 > maxIpCount {
		return false
	}
	//if common.DivideInt64(m.downgradeCount+1, m.count) >= maxIpRatio {
	//	return false
	//}

	key := snapshot.GetInvocationKey()
	if value, ok := m.stats.Load(key); ok {
		stat := value.(*InvocationStat)
		strategy := fault_tolerance_strategy.GetStrategy()
		if strategy == constant.FAULT_TOLERANCE_STRATEGY_ON {
			stat.Downgrade()
			m.downgradeCount++
			if tolerance_log.FaultToleranceLog.GetLogLevel() >= log.INFO {
				tolerance_log.FaultToleranceLog.Infof("downgrade a address. stat = %v.", stat)
			}
			return true
		} else if strategy == constant.FAULT_TOLERANCE_STRATEGY_MONITOR {
			if tolerance_log.FaultToleranceLog.GetLogLevel() >= log.INFO {
				tolerance_log.FaultToleranceLog.Infof("monitor. downgrade a address. stat = %v.", stat)
			}
			return false
		} else {
			return false
		}
	}
	return false
}

func (m *MeasureModel) snapshotInvocations() []*InvocationStat {
	snapshots := []*InvocationStat{}
	m.stats.Range(func(app, value interface{}) bool {
		stat := value.(*InvocationStat)
		//
		if !stat.IsHealthy() {
			m.recover(stat)
			return true
		}
		//
		if stat.GetCall() <= 0 {
			if stat.AddUselessCycle() {
				m.releaseInvocationStat(stat)
			}
			return true
		} else {
			stat.RestUselessCycle()
		}
		//
		snapshot := value.(*InvocationStat).Snapshot()
		snapshots = append(snapshots, snapshot)
		return true
	})
	return snapshots
}

func (m *MeasureModel) recover(stat *InvocationStat) {
	if downgradeTime := stat.downgradeTime; downgradeTime != 0 {
		now := common.GetNowMS()
		if now-stat.downgradeTime >= fault_tolerance_strategy.GetRecoveryTime() {
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
	var sumException int64
	var sumCall int64
	for _, stat := range stats {
		if call, exception := stat.GetCount(); call >= leastWindowCount {
			sumException += exception
			sumCall += call
		}
	}
	if sumCall == 0 {
		return false, 0
	}
	return true, common.DivideInt64(sumException, sumCall)
}

func (m *MeasureModel) IsArrivalTime(config *fault_tolerance_rule.FaultToleranceRule) bool {
	timeWindow := config.GetTimeWindow()
	now := common.GetNowMS()

	if m.timeMeter == 0 {
		m.timeMeter = now + timeWindow
		return false
	} else {
		if now >= m.timeMeter {
			m.timeMeter = now + timeWindow
			if tolerance_log.FaultToleranceLog.GetLogLevel() >= log.DEBUG {
				tolerance_log.FaultToleranceLog.Debugf("[Tolerance][MeasureModel][%s] arrivalTime, measureModel = %v, config = %v", m.key, m, config)
			}
			return true
		} else {
			return false
		}
	}
}

func (m *MeasureModel) String() string {
	str := fmt.Sprintf("key=%s,count=%v,downgradeCount=%v,timeMeter=%v",
		m.key, m.count, m.downgradeCount, m.timeMeter)
	return str
}
