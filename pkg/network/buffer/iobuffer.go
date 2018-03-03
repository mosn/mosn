package buffer

import (
	"io"
	"errors"
)

var (
	ErrBufFull           = errors.New("io buffer: buffer is full")
	ErrNegativeCount     = errors.New("io buffer: negative count")
	ErrInvalidWriteCount = errors.New("io buffer: invalid write count")
)

// IoBuffer
type IoBuffer struct {
	buf []byte // contents: buf[off : len(buf)]
	off int    // read from &buf[off], write to &buf[len(buf)]
}

func (b *IoBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	if b.off >= len(b.buf) {
		b.Reset()
	}

	for {
		if cap(b.buf)-len(b.buf) == 0 {
			return n, ErrBufFull
		}

		m, e := r.Read(b.buf[len(b.buf):cap(b.buf)])
		b.buf = b.buf[0: len(b.buf)+m]
		n += int64(m)

		if e != nil {
			return n, e
		}

		if m == 0 {
			return n, e
		}
	}

	return n, nil
}

func (b *IoBuffer) WriteTo(w io.Writer) (n int64, err error) {
	for b.off < len(b.buf) {
		nBytes := b.Len()
		m, e := w.Write(b.buf[b.off:])

		if m > nBytes {
			panic(ErrInvalidWriteCount)
		}

		b.off += m
		n += int64(m)

		if e != nil {
			return n, e
		}
	}

	return
}

func (b *IoBuffer) Bytes() []byte {
	return b.buf[b.off:]
}

func (b *IoBuffer) String() string {
	return string(b.buf[b.off:])
}

func (b *IoBuffer) Len() int {
	return len(b.buf) - b.off
}

func (b *IoBuffer) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
}

func (b *IoBuffer) available() int {
	return len(b.buf) - b.off
}

func NewIoBuffer(bufSize int) *IoBuffer {
	buf := make([]byte, 0, bufSize)

	return &IoBuffer{buf: buf}
}
