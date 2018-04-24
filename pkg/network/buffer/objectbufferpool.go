package buffer

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"sync"
)

type objectPool struct {
	sync.Pool
}

func (p *objectPool) Take() (object interface{}) {
	object = p.Get()

	return
}

func (p *objectPool) Give(object interface{}) {
	p.Put(object)
}

func NewObjectPool(poolSize int) types.ObjectBufferPool {
	pool := &objectPool{}

	return pool
}
