package buffer

import (
	"io"
	"io/ioutil"
	"bufio"
)

type ReaderBufferPool struct {
	bufSize int
	pool    chan *ReaderBufferPoolEntry
}

type ReaderBufferPoolEntry struct {
	Br     *bufio.Reader
	source io.Reader
}

func (bpe *ReaderBufferPoolEntry) Read(p []byte) (n int, err error) {
	return bpe.source.Read(p)
}

func (p *ReaderBufferPool) Take(r io.Reader) (bpe *ReaderBufferPoolEntry) {
	select {
	case bpe = <-p.pool:
		// prepare for reuse
		if a := bpe.Br.Buffered(); a > 0 {
			// drain the internal buffer
			io.CopyN(ioutil.Discard, bpe.Br, int64(a))
		}
		// swap out the underlying reader
		bpe.source = r
	default:
		// none available.  create a new one
		bpe = &ReaderBufferPoolEntry{nil, r}
		bpe.Br = bufio.NewReaderSize(bpe, p.bufSize)
	}

	return
}

func (p *ReaderBufferPool) Give(bpe *ReaderBufferPoolEntry) {
	select {
	case p.pool <- bpe: // return to pool
	default: // discard
	}
}

func NewReadBufferPool(poolSize, bufferSize int) *ReaderBufferPool {
	return &ReaderBufferPool{
		bufSize: bufferSize,
		pool:    make(chan *ReaderBufferPoolEntry, poolSize),
	}
}

type WriteBufferPool struct {
	bufSize int
	pool    chan *WriteBufferPoolEntry
}

type WriteBufferPoolEntry struct {
	Br     *bufio.Writer
	source io.Writer
}

func (bpe *WriteBufferPoolEntry) Write(p []byte) (n int, err error) {
	return bpe.source.Write(p)
}

func (p *WriteBufferPool) Take(r io.Writer) (bpe *WriteBufferPoolEntry) {
	select {
	case bpe = <-p.pool:
		bpe.source = r
	default:
		// none available.  create a new one
		bpe = &WriteBufferPoolEntry{nil, r}
		bpe.Br = bufio.NewWriterSize(bpe, p.bufSize)
	}
	return
}

func (p *WriteBufferPool) Give(bpe *WriteBufferPoolEntry) {
	if bpe.Br.Buffered() > 0 {
		return
	}
	if err := bpe.Br.Flush(); err != nil {
		return
	}
	select {
	case p.pool <- bpe: // return to pool
	default: // discard
	}
}

func NewWriteBufferPool(poolSize, bufferSize int) *WriteBufferPool {
	return &WriteBufferPool{
		bufSize: bufferSize,
		pool:    make(chan *WriteBufferPoolEntry, poolSize),
	}
}
