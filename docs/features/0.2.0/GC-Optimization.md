# 0.2.0 GC 与内存使用优化

##  整体方案

+ 新增内存复用框架，可以简单复用自定义内存
+ 改进BytePool， 更加方便全局复用
+ 改进IoBufferPool，底层使用BytePool，提高内存复用度
+ 改进Read/Write Buffer，减少内存使用，提升性能
+ 优化sofa部分实现，减少临时对象生成
+ 复用protocol， stream， proxy层的常用内存结构

## 细节与变更

### 内存复用框架
通过实现如下接口，可方便复用所需内存
+ Name() 返回自定义内存结构的唯一标识。
+ New() 新建内存结构的方法
+ Reset(）重置内存结构的方法，用于回收前调用，清空结构。
```go
// BufferPoolCtx is the bufferpool's context
type BufferPoolCtx interface {
	// Name returns the bufferpool's name
	Name() int

	// New returns the buffer
	New() interface{}

	// Reset resets the buffer
	Reset(interface{})
}
```
以Sofa为例:
+ `SofaProtocolBufferCtx`实现了`BufferPoolCtx`接口
+ `SofaProtocolBuffers` 为需要复用的内存结构体
+ `SofaProtocolBuffersByContent` 方式用于获取内存结构体
+ 内存结构体会在请求结束之后统一释放，请求上下文通过 context 关联
```go
type SofaProtocolBufferCtx struct{}

func (ctx SofaProtocolBufferCtx) Name() int {
	return buffer.SofaProtocol
}

func (ctx SofaProtocolBufferCtx) New() interface{} {
	buffer := new(SofaProtocolBuffers)
	return buffer
}

func (ctx SofaProtocolBufferCtx) Reset(i interface{}) {
	buf, _ := i.(*SofaProtocolBuffers)
	buf.BoltReq = BoltRequestCommand{}
	buf.BoltRsp = BoltResponseCommand{}
	buf.BoltEncodeReq = BoltRequestCommand{}
	buf.BoltEncodeRsp = BoltResponseCommand{}
}

type SofaProtocolBuffers struct {
	BoltReq       BoltRequestCommand
	BoltRsp       BoltResponseCommand
	BoltEncodeReq BoltRequestCommand
	BoltEncodeRsp BoltResponseCommand
}

func SofaProtocolBuffersByContent(context context.Context) *SofaProtocolBuffers {
	ctx := buffer.PoolContext(context)
	return ctx.Find(SofaProtocolBufferCtx{}, nil).(*SofaProtocolBuffers)
}
```
使用方法：
```go
sofabuffers := sofarpc.SofaProtocolBuffersByContent(context)
request := &sofabuffers.BoltReq
```

### BytePool改进
新增`GetBytes()`和`PutBytes()`函数，用于全局获取和回收 *[]byte， 屏蔽pool实现细节。  
```go
// GetBytes returns *[]byte from byteBufferPool
func GetBytes(size int) *[]byte 

// PutBytes Put *[]byte to byteBufferPool
func PutBytes(buf *[]byte) 
```

### IoBuffer改进
+ 新增`GetIoBuffer()`和`PutIoBuffer()`函数，用于全局获取和释放 types.IoBuffer，屏蔽pool实现细节。
```go
// GetIoBuffer returns IoBuffer from pool
func GetIoBuffer(size int) types.IoBuffer

// PutIoBuffer returns IoBuffer to pool
func PutIoBuffer(buf types.IoBuffer)
```

+ Iobuffer封装的 []byte 使用`GetBytes()`获取。
```go
func NewIoBuffer(capacity int) types.IoBuffer {
	buffer := &IoBuffer{
		offMark: ResetOffMark,
	}
	if capacity <= 0 {
		capacity = DefaultSize
	}
	buffer.b = GetBytes(capacity)
	buffer.buf = (*buffer.b)[:0]
	return buffer
}
```
+ 新增方法`Free()`用于释放[]byte,  `Alloc()`用于分配[]byte

```go
func (b *IoBuffer) Free() 

func (b *IoBuffer) Alloc(size int)
```

### Read/Write buffer
##### Read buffer
1. 循环读取，直到不可读或者达到预设的最大值, IoBuffer的扩容使用了`GetBytes()`和`PutBytes()`方法，提高内存复用。
2. 如果遇到读超时，调用`Free()`释放[]byte，然后调用`Alloc()`分配一个1字节的[]byte，用于等待读事件，减少空闲连接的内存使用。
  ```go
	if te, ok := err.(net.Error); ok && te.Timeout() {
		if c.readBuffer != nil && c.readBuffer.Len() == 0 {
			c.readBuffer.Free()
			c.readBuffer.Alloc(DefaultBufferReadCapacity)
		}
		continue
	}	
  ```
 
 ##### Write buffer
 优化之前的write调用为writev，减少内存分配和拷贝，减少锁力度。
 1. write将通过chan把iobuffer的切片指针传递给wirteloop协程
 ```go
 func (c *connection) Write(buffers ...types.IoBuffer) error {

	c.writeBufferChan <- &buffers

	return nil
}
 ```
 2. wirteLoop协程合并IoBuffer， 然后调用`doWriteIo`
 ```go
 func (c *connection) startWriteLoop() {
		select {
		case buf := <-c.writeBufferChan:
			c.appendBuffer(buf)

			//todo: dynamic set loop nums
			for i := 0; i < 10; i++ {
				select {
				case buf := <-c.writeBufferChan:
					c.appendBuffer(buf)
				default:
				}
			}
			_, err = c.doWriteIo()
		}
 ```
 3. 调用net.Buffers的`WriteTo`方法，使用wirtev系统调用一次发送多个IoBuffer
 4. 最后调用`PutIoBuffer()`， 释放已经发送的IoBuffer
 ```go
func (c *connection) doWriteIo() (bytesSent int64, err error) {
	bytesSent, err = c.writeBuffers.WriteTo(c.rawConnection)
	if err != nil {
		return bytesSent, err
	}
	for _, buf := range c.ioBuffers {
		buffer.PutIoBuffer(buf)
	}
	c.ioBuffers = c.ioBuffers[:0]
	return
}
 ```
* buffersWriter接口
```go
// buffersWriter is the interface implemented by Conns that support a
// "writev"-like batch write optimization.
// writeBuffers should fully consume and write all chunks from the
// provided Buffers, else it should report a non-nil error.
type buffersWriter interface {
	writeBuffers(*Buffers) (int64, error)
}
```
### sofa优化
* 使用`SofaProtocolBuffers`复用结构体，复用IoBuffer，map等结构。
* 减少临时对象   
   `ConvertPropertyValue`返回值为interface{}, 会产生临时对象，该方法在解析解析过程中调用量很大，通过`ConvertPropertyValueUint8`等系列函数规避临时对象产生。
 ```go
 func ConvertPropertyValue(strValue string, kind reflect.Kind) interface{} {
	switch kind {
	case reflect.Uint8:
		value, _ := strconv.ParseUint(strValue, 10, 8)
		return byte(value)
	case reflect.Uint16:
		value, _ := strconv.ParseUint(strValue, 10, 16)
		return uint16(value)
 }	
 
 func ConvertPropertyValueUint8(strValue string) byte {
	value, _ := strconv.ParseUint(strValue, 10, 8)
	return byte(value)
 }

 func ConvertPropertyValueUint16(strValue string) uint16 {
	value, _ := strconv.ParseUint(strValue, 10, 16)
	return uint16(value)
 }
 ```

## 测试
### 单元测试 unit-test
测试代码：buffer_test.go
* Bench (场景供参考）
```go
goos: darwin
goarch: amd64
pkg: github.com/alipay/sofa-mosn/pkg/buffer
// 使用 BytePool分配
1000000	      1762 ns/op	      26 B/op	       0 allocs/op
// 直接分配 []byte 
1000000	      2086 ns/op	    2048 B/op	       1 allocs/op
// 使用 IobufferPool 分配
1000000	      1225 ns/op	      31 B/op	       0 allocs/op
// 直接分配 Iobuffer
1000000	      1585 ns/op	    2133 B/op	       3 allocs/op
PASS
```

