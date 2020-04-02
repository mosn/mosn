## 自定义结构体复用

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
