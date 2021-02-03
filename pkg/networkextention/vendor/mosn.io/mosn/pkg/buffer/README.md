## 自定义结构体复用
##### 请求维度的内存申请复用
* 模板
```
package example

import (
	"context"

	"mosn.io/mosn/pkg/buffer"
	"net/http"
)

var ins exampleBufferCtx

// 注册buffer类型到内存复用框架
func init() {
	buffer.RegisterBuffer(&ins)
}

// 需要包含 buffer.TempBufferCtx 到自定义的Ctx, 且要放到第一位
type exampleBufferCtx struct{
	buffer.TempBufferCtx
}

// 实现New()函数， 用于生成自定义buffer
func (ctx exampleBufferCtx) New() interface{} {
	buffer := new(exampleBuffers)
	return buffer
}

// 实现Reset()函数， 用于回收buffer之前，重置buffer内复用的结构体
func (ctx exampleBufferCtx) Reset(i interface{}) {
	buf := i.(*exampleBufferCtx)
	*buf = exampleBufferCtx{}
}

// 自定义buffer结构体，包含需要复用的结构体
type exampleBuffers struct {
	req http.Request
	rsp http.Response
}

// 通过ctx获取复用buffer
func exampleBuffersByContext(ctx context.Context) *exampleBuffers {
	poolCtx := buffer.PoolContext(ctx)
	return poolCtx.Find(&ins, nil).(*exampleBuffers)
}
```
* 使用方式
```
func run(ctx context.Context) {
    // 通过ctx获取内存块
	buffer := exampleBuffersByContext(ctx)
	// 通过指针使用
	req := &buffer.req
	rsp := &buffer.rsp
}
```

## IoBuffer复用
```
// GetIoBuffer returns IoBuffer from pool
func GetIoBuffer(size int) types.IoBuffer {
	return ibPool.take(size)
}

// PutIoBuffer returns IoBuffer to pool
func PutIoBuffer(buf types.IoBuffer) {
	if buf.Count(-1) != 0 {
		return
	}
	ibPool.give(buf)
}
```


## Byte复用
```
// GetBytes returns *[]byte from byteBufferPool
func GetBytes(size int) *[]byte {
	p := getByteBufferPool()
	return p.take(size)
}

// PutBytes Put *[]byte to byteBufferPool
func PutBytes(buf *[]byte) {
	p := getByteBufferPool()
	p.give(buf)
}
```
