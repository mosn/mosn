package buffer

import (
	"sync"

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type headersBufferPoolV2 struct {
	sync.Pool
}

func (p *headersBufferPoolV2) Take(defaultSize int) (amap map[string]string) {
	v := p.Get()

	if v == nil {
		amap = make(map[string]string, defaultSize)
	} else {
		amap = v.(map[string]string)
	}

	return
}

func (p *headersBufferPoolV2) Give(amap map[string]string) {
	for k := range amap {
		delete(amap, k)
	}

	p.Put(amap)
}

func NewHeadersBufferPool(poolSize int) types.HeadersBufferPool {
	return &headersBufferPoolV2{}
}
