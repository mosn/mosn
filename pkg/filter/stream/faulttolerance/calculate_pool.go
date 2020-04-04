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

}

func (p *CalculatePool) GetRegulationModel(invocationStatDimension InvocationStatDimension) *MeasureModel {
	key := invocationStatDimension.GetInvocationKey()
	if value, ok := p.appRegulationModels.Load(key); ok {
		return value.(*MeasureModel)
	} else {
		measureModel := NewMeasureModel(invocationStatDimension.GetMeasureKey())
		measureModel.addInvocationStat(invocationStatDimension)
		value, ok := p.appRegulationModels.LoadOrStore(key, measureModel)
		if ok {
			value.(*MeasureModel).addInvocationStat(invocationStatDimension)
		}
		return value.(*MeasureModel)
	}
}
