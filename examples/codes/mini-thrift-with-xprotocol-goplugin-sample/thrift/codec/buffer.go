/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"

	"mosn.io/pkg/buffer"
)

// ContextKey type
type ContextKey int

// Context key types(built-in)
const (
	ContextKeyStreamID ContextKey = iota
	ContextKeyConnection
	ContextKeyConnectionID
	ContextKeyConnectionPoolIndex
	ContextKeyListenerPort
	ContextKeyListenerName
	ContextKeyListenerType
	ContextKeyListenerStatsNameSpace
	ContextKeyNetworkFilterChainFactories
	ContextKeyStreamFilterChainFactories
	ContextKeyBufferPoolCtx
	ContextKeyAccessLogs
	ContextOriRemoteAddr
	ContextKeyAcceptChan
	ContextKeyAcceptBuffer
	ContextKeyConnectionFd
	ContextSubProtocol
	ContextKeyTraceSpanKey
	ContextKeyActiveSpan
	ContextKeyTraceId
	ContextKeyVariables
	ContextKeyProxyGeneralConfig
	ContextKeyDownStreamProtocol
	ContextKeyConfigDownStreamProtocol
	ContextKeyConfigUpStreamProtocol
	ContextKeyDownStreamHeaders
	ContextKeyDownStreamRespHeaders
	ContextKeyEnd
)

// GlobalProxyName represents proxy name for metrics
const (
	GlobalProxyName       = "global"
	GlobalShutdownTimeout = "GlobalShutdownTimeout"
)

type valueCtx struct {
	context.Context

	builtin [ContextKeyEnd]interface{}
}

func (c *valueCtx) Value(key interface{}) interface{} {
	if contextKey, ok := key.(ContextKey); ok {
		return c.builtin[contextKey]
	}
	return c.Context.Value(key)
}

func Get(ctx context.Context, key ContextKey) interface{} {
	if mosnCtx, ok := ctx.(*valueCtx); ok {
		return mosnCtx.builtin[key]
	}
	return ctx.Value(key)
}

// WithValue add the given key-value pair into the existed value context, or create a new value context which contains the pair.
// This Function should not be used along with the official context.WithValue !!

// The following context topology will leads to existed pair {'foo':'bar'} NOT FOUND, because recursive lookup for
// key-type=types.ContextKey is not supported by mosn.valueCtx.
//
// topology: context.Background -> mosn.valueCtx{'foo':'bar'} -> context.valueCtx -> mosn.valueCtx{'hmm':'haa'}
func WithValue(parent context.Context, key ContextKey, value interface{}) context.Context {
	if mosnCtx, ok := parent.(*valueCtx); ok {
		mosnCtx.builtin[key] = value
		return mosnCtx
	}

	// create new valueCtx
	mosnCtx := &valueCtx{Context: parent}
	mosnCtx.builtin[key] = value
	return mosnCtx
}

// Clone copy the origin mosn value context(if it is), and return new one
func Clone(parent context.Context) context.Context {
	if mosnCtx, ok := parent.(*valueCtx); ok {
		clone := &valueCtx{Context: mosnCtx}
		// array copy assign
		clone.builtin = mosnCtx.builtin
		return clone
	}
	return parent
}

type BufferPoolCtx interface {
	// Index returns the bufferpool's Index
	Index() int

	// New returns the buffer
	New() interface{}

	// Reset resets the buffer
	Reset(interface{})
}

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
	return WithValue(ctx, ContextKeyBufferPoolCtx, newBufferValue())
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
		if val := Get(ctx, ContextKeyBufferPoolCtx); val != nil {
			return val.(*bufferValue)
		}
	}
	return newBufferValue()
}

var bufferCtx = ByteBufferCtx{}

func init() {
	RegisterBuffer(&bufferCtx)
	RegisterBuffer(&ins)
}

type ByteBufferCtx struct {
	TempBufferCtx
}

func (ctx ByteBufferCtx) New() interface{} {
	return buffer.NewByteBufferPoolContainer()
}

func (ctx ByteBufferCtx) Reset(i interface{}) {
	p := i.(*buffer.ByteBufferPoolContainer)
	p.Reset()
}

// GetBytesByContext returns []byte from byteBufferPool by context
func GetBytesByContext(context context.Context, size int) *[]byte {
	p := PoolContext(context).Find(&bufferCtx, nil).(*buffer.ByteBufferPoolContainer)
	return p.Take(size)
}

var ins thriftBufferCtx

type thriftBufferCtx struct {
	TempBufferCtx
}

func (ctx thriftBufferCtx) New() interface{} {
	return new(thriftBuffer)
}

func (ctx thriftBufferCtx) Reset(i interface{}) {
	buf, _ := i.(*thriftBuffer)

	// recycle ioBuffer
	if buf.request.Data != nil {
		if e := buffer.PutIoBuffer(buf.request.Data); e != nil {
			//log.DefaultLogger.Errorf("[protocol] [thrift] [buffer] [reset] PutIoBuffer error: %v", e)
		}
	}

	if buf.response.Data != nil {
		if e := buffer.PutIoBuffer(buf.response.Data); e != nil {
			//log.DefaultLogger.Errorf("[protocol] [thrift] [buffer] [reset] PutIoBuffer error: %v", e)
		}
	}

	*buf = thriftBuffer{}
}

type thriftBuffer struct {
	request  Request
	response Response
}

func bufferByContext(ctx context.Context) *thriftBuffer {
	poolCtx := PoolContext(ctx)
	return poolCtx.Find(&ins, nil).(*thriftBuffer)
}
