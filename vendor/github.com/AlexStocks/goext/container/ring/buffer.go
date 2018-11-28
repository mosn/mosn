package gxring

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
)

// buffer file format related sizes
const (
	// head
	size     = 4 // bytes in frame header used to store size of frame
	sequence = 8 // bytes in frame header used to store sequence of record
	total    = size + sequence

	// tail
	metadata = 4 << 10 // bytes in file at the end reserved for metadata
)

var maxBytes uint32 = (1 << (uint(size) * 8)) - total // the biggest size of data a frame can store

type frame []byte

var (
	errFrameTooBig = fmt.Errorf("buffer: data more than max %d bytes", maxBytes)
	errDataTooBig  = errors.New("buffer: data overflows buffer")
)

func newFrame(seq uint64, data []byte) (frame, error) {
	if len(data) > int(maxBytes) {
		return nil, errFrameTooBig
	}
	var f frame = make([]byte, len(data)+total)
	binary.PutLittleEndianUint32(f, 0, uint32(len(f)))
	binary.PutLittleEndianUint64(f, size, seq)
	copy(f[total:], data)
	return f, nil
}

// size returns the size of frame encoded in the frame.
// If the complete frame was read it'd match the length.
// It returns 0 if not enough of the frame has been read
// to determine the encoded size.
func (f frame) size() uint32 {
	if len(f) < size {
		return 0
	}
	return binary.GetLittleEndianUint32(f, 0)
}

// seq returns the record sequence encoded in the frame.
// It returns 0 if not enough of the frame has been read
// to determine the encoded sequence.
func (f frame) seq() uint64 {
	if len(f) < total {
		return 0
	}
	return binary.GetLittleEndianUint64(f, size)
}

// data returns the record data stored in the frame.
// It returns nil if not enough of the frame has been
// read to read at least one byte of data.
//
// The underlying array of the slice returned may be
// recycled.
func (f frame) data() []byte {
	if len(f) < total {
		return nil
	}
	return f[total:]
}

type Buffer struct {
	capacity uint64
	nextSeq  uint64
	biggest  uint32 // largest frame seen, doesn't reset but helps frameOfLen
	length   uint64 // total data stored

	// These are start offsets for first and
	// last records maintained by the buffer.
	// They are abosulte and not wrapped.
	first uint64
	last  uint64

	filename string
	data     []byte

	mu sync.RWMutex // protects whole Buffer
}

func New(capacity int, filename string) (*Buffer, error) {
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return nil, ErrFileExists
	}

	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fsize := capacity + metadata
	if err := syscall.Truncate(filename, int64(fsize)); err != nil {
		return nil, err
	}

	data, err := syscall.Mmap(
		int(f.Fd()), 0, fsize,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED,
	)
	// TODO(ashish): Call msync periodically.
	if err != nil {
		return nil, err
	}

	b := &Buffer{
		capacity: uint64(capacity), // since it's used with uint64s a lot more
		filename: f.Name(),
		data:     data,
	}
	b.updateMeta()

	return b, nil
}

// Len returns the total data bytes stored in the buffer.
// This does not include the space used for framing.
func (b *Buffer) Len() uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.length
}

// Remaining returns the total bytes remaining to be read
// before reading everything in the buffer.
func (b *Buffer) Remaining(from Cursor) uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if from.offset > b.last {
		return 0
	}

	off := from.offset
	if off < b.first {
		off = b.first
	}
	remaining := b.last - off
	remaining += uint64(b.frameSize(b.last))
	return remaining
}

// NOTE: No memlock because do not want to deal with RLIMIT_MEMLOCK
// on the edge boxes yet.

func (b *Buffer) updateMeta() {
	// First 8 bytes have the first frame offset,
	// next 8 bytes have the last frame offset,
	// next 8 bytes are the next sequence number,
	// next 4 bytes are the biggest data record we've seen,
	// next 8 bytes are the total data in the buffer.
	off := int(b.capacity)
	binary.PutLittleEndianUint64(b.data, off, b.first)
	binary.PutLittleEndianUint64(b.data, off+8, b.last)
	binary.PutLittleEndianUint64(b.data, off+16, b.nextSeq)
	binary.PutLittleEndianUint32(b.data, off+24, b.biggest)
	binary.PutLittleEndianUint64(b.data, off+28, b.length)
}

func Load(filename string) (*Buffer, error) {
	b := &Buffer{filename: filename}
	err := b.reload()
	return b, err
}

// reload reads back info about b from b.filename.
// b.filename should be set.
func (b *Buffer) reload() error {
	stat, err := os.Stat(b.filename)
	if err != nil {
		return err
	}
	fsize := uint64(stat.Size())
	b.capacity = fsize - metadata

	f, err := os.OpenFile(b.filename, os.O_RDWR, 0600)
	if err != nil {
		return err
	}

	data, err := syscall.Mmap(
		int(f.Fd()), 0, int(fsize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED,
	)
	if err != nil {
		return err
	}
	b.data = data

	off := int(b.capacity)
	b.first = binary.GetLittleEndianUint64(b.data, off)
	b.last = binary.GetLittleEndianUint64(b.data, off+8)
	b.nextSeq = binary.GetLittleEndianUint64(b.data, off+16)
	b.biggest = binary.GetLittleEndianUint32(b.data, off+24)
	b.length = binary.GetLittleEndianUint64(b.data, off+28)

	return nil
}

// Insert adds data into b. Caller is free to recycle the underlying
// array of the slice.
func (b *Buffer) Insert(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	frame, err := newFrame(b.nextSeq, data) // can be avoided
	if err != nil {
		return err
	}

	fsize := frame.size()
	fsize64 := uint64(fsize)
	if fsize64 > b.capacity {
		return errDataTooBig
	}

	b.last = b.nextFrameOffset(b.last) // new last where we'd start writing frame
	b.updateFirst(fsize64)

	start := b.last % b.capacity
	end := start + fsize64

	var wrap bool
	if end > b.capacity {
		wrap = true
		end = b.capacity
	}
	copyoff := end - start
	copy(b.data[start:end], frame[:copyoff])
	if wrap {
		copy(b.data, frame[copyoff:])
	}

	if fsize > b.biggest {
		b.biggest = fsize
	}
	b.nextSeq++
	b.length += uint64(len(data))
	b.updateMeta()
	return nil
}

// updateFirst updates the offset of the first record in b.
//
// This should be called before updating b.biggest or writing any data but
// after updating b.last to the value where the new record would be written.
func (b *Buffer) updateFirst(fsize uint64) {
	if b.biggest == 0 {
		// Just starting out, no need to update.
		return
	}

	var (
		start    = b.last % b.capacity
		end      = (start + fsize) % b.capacity
		wrapping = end <= start
	)

	if start == end {
		b.length = 0
		b.first = b.last
		return
	}

	for {
		if b.first == b.last {
			// b can fit only the new incoming record.
			return
		}

		firstWrapped := b.first % b.capacity

		if wrapping {
			if end <= firstWrapped && firstWrapped < start {
				return
			}
		} else {
			if end <= firstWrapped {
				return
			}
			if start > firstWrapped {
				return
			}
		}

		second := b.nextFrameOffset(b.first)
		b.length -= (second - b.first - total)
		b.first = second

		// May need to discard multiple records at the begining.
	}
}

// nextFrameOffset returns the offset from where the next frame
// should start. It is an absolute offset and not wrapped around.
func (b *Buffer) nextFrameOffset(offset uint64) uint64 {
	return offset + uint64(b.frameSize(offset))
}

// frame returns the frame encoded at offset.
func (b *Buffer) frame(offset uint64) frame {
	return b.frameOfLen(offset, b.frameSize(offset))
}

// frameSize returns the size of the frame encoded at offset.
func (b *Buffer) frameSize(offset uint64) uint32 {
	f := b.frameOfLen(offset, size)
	return f.size()
}

// frameOfLen returns a frame of length starting at offset. The size
// of frame may differ from the length.
func (b *Buffer) frameOfLen(offset uint64, length uint32) frame {
	if length > b.biggest {
		// This can happen, say, when a non-header region is
		// interpreted as a header region and may have junk
		// frame length.
		return nil
	}

	var f frame

	start := offset % b.capacity
	end := start + uint64(length)

	if end > b.capacity {
		// The record wraps around. Until we mmap the file back-to-back
		// we cannot slice b.data and need to copy the wrapped contents.
		f = make([]byte, length)
		copy(f, b.data[start:b.capacity])
		copy(f[b.capacity-start:], b.data[:end%b.capacity])
	} else {
		f = b.data[start:end]
	}
	return f
}

var (
	ErrFileExists  = errors.New("buffer: log file already exists")
	ErrPartialData = errors.New("buffer: partial data returned")
	ErrNotArrived  = errors.New("buffer: data has not yet arrived")
)

// Read copies record that c points to into data. It returns number of bytes copied,
// a cursor to the next record, and error if there was any.
//
// If c points to a record that has been ejected from b the call is analogous to
// b.ReadFirst.
//
// If c points to a record that has not yet arrived an ErrNotArrived error is returned.
//
// If data does not have enough room for record an ErrPartialData error is returned.
func (b *Buffer) Read(data []byte, c Cursor) (n int, next Cursor, err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	seq, offset := c.seq, c.offset

	if seq >= b.nextSeq || offset > b.last {
		return 0, next, ErrNotArrived
	}

	f := b.frame(offset)
	if f.size() == 0 || f.seq() != seq {
		return b.readFirst(data)
	}

	return b.readOffset(data, offset)
}

// ReadFirst reads the first (oldest) record in b. It's return values are analogous
// to Read.
func (b *Buffer) ReadFirst(data []byte) (n int, next Cursor, err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.readOffset(data, b.first)
}

func (b *Buffer) readFirst(data []byte) (n int, next Cursor, err error) {
	return b.readOffset(data, b.first)
}

func (b *Buffer) readOffset(data []byte, offset uint64) (n int, next Cursor, err error) {
	if b.biggest == 0 {
		return 0, next, ErrNotArrived
	}

	f := b.frame(offset)
	d := f.data()
	n = copy(data, d)

	if n != len(d) {
		err = ErrPartialData
	}

	next.offset = b.nextFrameOffset(offset)

	if next.offset-offset == b.capacity {
		// There is only one frame in b.
		next.seq = b.nextSeq
		return
	}

	if offset == b.last {
		next.seq = b.nextSeq
	} else {
		nextf := b.frame(next.offset)
		next.seq = nextf.seq()
	}

	return
}

var pageSize uint64

func init() {
	pageSize = uint64(os.Getpagesize())
}

// Unmap unmaps the buffer file from memory.
func (b *Buffer) Unmap() {
	if err := syscall.Munmap(b.data); err != nil {
		// Munmap should only fail if we pass it a bad pointer
		// which should not happen. If it does something has
		// gone terribly wrong and should not proceed further.
		panic(err)
	}
}

// Close calls b.Unmap and deletes the buffer file.
func (b *Buffer) Close() error {
	b.Unmap()
	return os.Remove(b.filename)
}

type Cursor struct {
	offset uint64
	seq    uint64
}

// Seq returns the sequence number of the record c points
// to in Buffer. Sequence numbers are sequential.
func (c Cursor) Seq() uint64 {
	return c.seq
}
