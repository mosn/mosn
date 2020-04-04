package faulttolerance

import (
	"sync"
)

type CalculatePool struct {
	invocationStats     *sync.Map
	appRegulationModels *sync.Map
}

func NewCalculatePool() *CalculatePool {
	return &CalculatePool{}
}

func (p *CalculatePool) Regulate(dimension InvocationStatDimension) {
	if measureModel := p.GetRegulationModel(dimension); measureModel != nil {

	}
}

func (p *CalculatePool) GetRegulationModel(invocationStatDimension InvocationStatDimension) *MeasureModel {
	key := invocationStatDimension.GetInvocationKey()
	if value, ok := p.appRegulationModels.Load(key); ok {
		value.(*MeasureModel).addInvocationStat(invocationStatDimension)
		return nil
	} else {
		measureModel := NewMeasureModel(invocationStatDimension.GetMeasureKey())
		measureModel.addInvocationStat(invocationStatDimension)
		if value, ok := p.appRegulationModels.LoadOrStore(key, measureModel); ok {
			value.(*MeasureModel).addInvocationStat(invocationStatDimension)
			return nil
		} else {
			return value.(*MeasureModel)
		}
	}
}
