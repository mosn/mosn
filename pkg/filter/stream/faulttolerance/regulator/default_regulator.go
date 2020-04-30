package regulator

import (
	"mosn.io/mosn/pkg/filter/stream/faulttolerance/invocation"
	"sync"
)

type DefaultRegulator struct {
	measureModels *sync.Map
	workPool      WorkPool
}

func NewDefaultRegulator() *DefaultRegulator {
	regulator := &DefaultRegulator{
		measureModels: new(sync.Map),
		workPool:      NewDefaultWorkPool(20),
	}
	return regulator
}

func (r *DefaultRegulator) Regulate(stat *invocation.InvocationStat) {
	if measureModel := r.createRegulationModel(stat); measureModel != nil {
		if tolerance_log.FaultToleranceLog.GetLogLevel() >= log.DEBUG {
			tolerance_log.FaultToleranceLog.Debugf("[Tolerance][InvocationRegulator] regulate a stat, create a regulation model. stat = %v, measureModel = %v", stat, measureModel)
		}
		r.workPool.Schedule(measureModel)
	} else {
		if tolerance_log.FaultToleranceLog.GetLogLevel() >= log.DEBUG {
			tolerance_log.FaultToleranceLog.Debugf("[Tolerance][InvocationRegulator] regulate a stat, exist regulation model. stat = %v, measureModel = %v", stat, measureModel)
		}
	}
}

func (r *DefaultRegulator) createRegulationModel(stat *invocation.InvocationStat) *MeasureModel {
	key := stat.GetMeasureKey()
	if value, ok := r.measureModels.Load(key); ok {
		value.(*MeasureModel).AddInvocationStat(stat)
		return nil
	} else {
		measureModel := NewMeasureModel(key)
		measureModel.AddInvocationStat(stat)
		if value, ok := r.measureModels.LoadOrStore(key, measureModel); ok {
			value.(*MeasureModel).AddInvocationStat(stat)
			return nil
		} else {
			return value.(*MeasureModel)
		}
	}
}
