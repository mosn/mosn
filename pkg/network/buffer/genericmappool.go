package buffer

import (
	"sync"

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type genericmapPoolV2 struct {
	sync.Pool
}

func (p *genericmapPoolV2) Take(defaultSize int) (amap map[string]interface{}) {
	v := p.Get()

	if v == nil {
		amap = make(map[string]interface{}, defaultSize)
	} else {
		amap = v.(map[string]interface{})
	}

	return
}

func (p *genericmapPoolV2) Give(amap map[string]interface{}) {
	for k := range amap {
		delete(amap, k)
	}

	p.Put(amap)
}

func NewGenericMapPool(poolSize int) types.GenericMapPool {
	//return &headersBufferPool{
	//	pool: make(chan map[string]string, poolSize),
	//}
	//
	return &genericmapPoolV2{}
}
