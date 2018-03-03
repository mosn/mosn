package buffer

import (
	"io"
)

type IoBufferPool struct {
	bufSize int
	pool    chan *IoBufferPoolEntry
}

type IoBufferPoolEntry struct {
	Br *IoBuffer
	io io.Reader
}

func (bpe *IoBufferPoolEntry) Read() (n int64, err error) {
	return bpe.Br.ReadFrom(bpe.io)
}

func (p *IoBufferPool) Take(r io.Reader) (bpe *IoBufferPoolEntry) {
	select {
	case bpe = <-p.pool:
		// swap out the underlying reader
		bpe.io = r
	default:
		// none available.  create a new one
		bpe = &IoBufferPoolEntry{nil, r}
		bpe.Br = NewIoBuffer(p.bufSize)
	}

	return
}

func (p *IoBufferPool) Give(bpe *IoBufferPoolEntry) {
	bpe.Br.Reset()

	select {
	case p.pool <- bpe: // return to pool
	default: // discard
	}
}

func NewIoBufferPool(poolSize, bufferSize int) *IoBufferPool {
	return &IoBufferPool{
		bufSize: bufferSize,
		pool:    make(chan *IoBufferPoolEntry, poolSize),
	}
}
