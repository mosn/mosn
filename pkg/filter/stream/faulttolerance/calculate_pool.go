package faulttolerance

import (
	"mosn.io/api"
)

type CalculatePool struct {
}

func NewCalculatePool() *CalculatePool {
	return &CalculatePool{}
}

func (p *CalculatePool) Regulate(headers api.HeaderMap) {

}

func (p *CalculatePool) GetInvocationStat(dimension InvocationStatDimension) InvocationStat {
	return InvocationStat{}
}
