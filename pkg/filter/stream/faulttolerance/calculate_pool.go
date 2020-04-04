package faulttolerance

import (
	"sync"
)

type CalculatePool struct {
	invocationStats *sync.Map
}

func NewCalculatePool() *CalculatePool {
	return &CalculatePool{}
}

func (p *CalculatePool) Regulate(stat InvocationStat) {

}
