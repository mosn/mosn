package faulttolerance

import (
	"sync"
	"time"
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
		tick := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-tick.C:
				p.doRegulate(measureModel)

			}
		}
	}
}

func (p *CalculatePool) doRegulate(measureModel *MeasureModel) {

}

func (p *CalculatePool) isArriveTimeWindow() {

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
