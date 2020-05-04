package regulator

import (
	"mosn.io/mosn/pkg/config/v2"
	"sync"
)

type DefaultRegulator struct {
	measureModels *sync.Map
	workPool      WorkPool
	config        *v2.FaultToleranceFilterConfig
}

func NewDefaultRegulator(config *v2.FaultToleranceFilterConfig) *DefaultRegulator {
	regulator := &DefaultRegulator{
		measureModels: new(sync.Map),
		workPool:      NewDefaultWorkPool(20),
		config:        config,
	}
	return regulator
}

func (r *DefaultRegulator) Regulate(stat *InvocationStat) {
	if measureModel := r.createRegulationModel(stat); measureModel != nil {
		r.workPool.Schedule(measureModel)
	}
}

func (r *DefaultRegulator) createRegulationModel(stat *InvocationStat) *MeasureModel {
	key := stat.GetMeasureKey()
	if value, ok := r.measureModels.Load(key); ok {
		value.(*MeasureModel).AddInvocationStat(stat)
		return nil
	} else {
		measureModel := NewMeasureModel(key, r.config)
		measureModel.AddInvocationStat(stat)
		if value, ok := r.measureModels.LoadOrStore(key, measureModel); ok {
			value.(*MeasureModel).AddInvocationStat(stat)
			return nil
		} else {
			return value.(*MeasureModel)
		}
	}
}
