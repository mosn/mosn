package faulttolerance

import (
	v2 "mosn.io/mosn/pkg/config/v2"
	"sync"
	"time"
)

type CalculatePool struct {
	config              *v2.FaultToleranceFilterConfig
	currentTimeWindow   uint32
	timeMeter           uint32
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
	if p.isArriveTimeWindow() {

	}
}

func (p *CalculatePool) measure(model MeasureModel) {
	dimensions := model.getInvocationDimensions()
	invocationStats := GetInvocationStatFactoryInstance().getSnapshotInvocationStats(dimensions)
	leastWindowCount := p.config.LeastWindowCount
	if leastWindowCount < LEGAL_LEAST_WINDOW_COUNT {
		leastWindowCount = LEGAL_LEAST_WINDOW_COUNT
	}

}

func (p *CalculatePool) calculateAverageExceptionRate(invocationStats []*InvocationStat, leastWindowCount uint64) (bool, float64) {
	var sumException uint64
	var sumCall uint64
	for _, stat := range invocationStats {
		if stat.GetCallCount() >= leastWindowCount {
			sumException += stat.GetExceptionCount()
			sumCall += stat.GetCallCount()
		}
	}
	if sumCall == 0 {
		return false, 0
	}

}

func (p *CalculatePool) isArriveTimeWindow() bool {
	timeWindow := p.config.TimeWindow
	if p.currentTimeWindow <= timeWindow {
		p.currentTimeWindow = timeWindow

		p.timeMeter++
		if p.timeMeter == p.currentTimeWindow {
			p.timeMeter = 0
			return true
		} else {
			return false
		}
	} else if p.timeMeter < timeWindow {
		p.currentTimeWindow = timeWindow

		p.timeMeter++

		if p.timeMeter == p.currentTimeWindow {
			p.timeMeter = 0
			return true
		} else {
			return false
		}
	} else {
		p.currentTimeWindow = timeWindow

		p.timeMeter = 0
		return true
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
