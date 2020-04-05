package faulttolerance

import (
	"sync"
)

type MeasureModel struct {
	key        string
	dimensions *sync.Map
}

func NewMeasureModel(key string) *MeasureModel {
	return &MeasureModel{
		key:        key,
		dimensions: new(sync.Map),
	}
}

func (m *MeasureModel) addInvocationStat(dimension InvocationStatDimension) {
	key := dimension.GetInvocationKey()
	m.dimensions.Store(key, dimension)
}

func (m *MeasureModel) getInvocationDimensions() *sync.Map {
	return m.dimensions
}
