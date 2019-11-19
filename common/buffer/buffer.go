package buffer

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"

	mosnctx "sofastack.io/sofa-mosn/common/context"
)

const maxBufferPool = 16

var (
	index int32
	bPool = bufferPoolArray[:]
	vPool = new(valuePool)

	bufferPoolArray [maxBufferPool]bufferPool
	nullBufferValue [maxBufferPool]interface{}
)

// TempBufferCtx is template for BufferPoolCtx
type TempBufferCtx struct {
	index int
}

func (t *TempBufferCtx) Index() int {
	return t.index
}

func (t *TempBufferCtx) New() interface{} {
	return nil
}

func (t *TempBufferCtx) Reset(x interface{}) {
}

// ifaceWords is interface internal representation.
type ifaceWords struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

// setIdex sets index, poolCtx must embedded TempBufferCtx
func setIndex(poolCtx BufferPoolCtx, i int) {
	p := (*ifaceWords)(unsafe.Pointer(&poolCtx))
	temp := (*TempBufferCtx)(p.data)
	temp.index = i
}

func RegisterBuffer(poolCtx BufferPoolCtx) {
	// frist index is 1
	i := atomic.AddInt32(&index, 1)
	if i >= maxBufferPool {
		panic("bufferSize over full")
	}
	bPool[i].ctx = poolCtx
	setIndex(poolCtx, int(i))
}

// bufferPool is buffer pool
type bufferPool struct {
	ctx BufferPoolCtx
	sync.Pool
}

type valuePool struct {
	sync.Pool
}

// Take returns a buffer from buffer pool
func (p *bufferPool) take() (value interface{}) {
	value = p.Get()
	if value == nil {
		value = p.ctx.New()
	}
	return
}

// Give returns a buffer to buffer pool
func (p *bufferPool) give(value interface{}) {
	p.ctx.Reset(value)
	p.Put(value)
}

// bufferValue is buffer pool's Value
type bufferValue struct {
	value    [maxBufferPool]interface{}
	transmit [maxBufferPool]interface{}
}

// NewBufferPoolContext returns a context with bufferValue
func NewBufferPoolContext(ctx context.Context) context.Context {
	return mosnctx.WithValue(ctx, mosnctx.ContextKeyBufferPoolCtx, newBufferValue())
}

// TransmitBufferPoolContext copy a context
func TransmitBufferPoolContext(dst context.Context, src context.Context) {
	sValue := PoolContext(src)
	if sValue.value == nullBufferValue {
		return
	}
	dValue := PoolContext(dst)
	dValue.transmit = sValue.value
	sValue.value = nullBufferValue
}

// newBufferValue returns bufferValue
func newBufferValue() (value *bufferValue) {
	v := vPool.Get()
	if v == nil {
		value = new(bufferValue)
	} else {
		value = v.(*bufferValue)
	}
	return
}

// Find returns buffer from bufferValue
func (bv *bufferValue) Find(poolCtx BufferPoolCtx, x interface{}) interface{} {
	i := poolCtx.Index()
	if i <= 0 || i > int(index) {
		panic("buffer should call buffer.RegisterBuffer()")
	}
	if bv.value[i] != nil {
		return bv.value[i]
	}
	return bv.Take(poolCtx)
}

// Take returns buffer from buffer pools
func (bv *bufferValue) Take(poolCtx BufferPoolCtx) (value interface{}) {
	i := poolCtx.Index()
	value = bPool[i].take()
	bv.value[i] = value
	return
}

// Give returns buffer to buffer pools
func (bv *bufferValue) Give() {
	if index <= 0 {
		return
	}
	// first index is 1
	for i := 1; i <= int(index); i++ {
		value := bv.value[i]
		if value != nil {
			bPool[i].give(value)
		}
		value = bv.transmit[i]
		if value != nil {
			bPool[i].give(value)
		}
	}
	bv.value = nullBufferValue
	bv.transmit = nullBufferValue

	// Give bufferValue to Pool
	vPool.Put(bv)
}

// PoolContext returns bufferValue by context
func PoolContext(ctx context.Context) *bufferValue {
	if ctx != nil {
		if val := mosnctx.Get(ctx, mosnctx.ContextKeyBufferPoolCtx); val != nil {
			return val.(*bufferValue)
		}
	}
	return newBufferValue()
}
