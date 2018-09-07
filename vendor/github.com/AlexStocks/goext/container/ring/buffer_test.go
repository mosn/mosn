package gxring

import (
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var bufFile = filepath.Join(os.TempDir(), "buffer-test")

func init() {
	os.Remove(bufFile)
}

func TestFrame(t *testing.T) {
	data := []byte("much data. such big. wow.")

	f, err := newFrame(0, data)
	if err != nil {
		t.Fatalf("Failed to create frame: %v", err)
	}
	fdata := f.data()

	if !bytes.Equal(fdata, data) {
		t.Errorf("f.data() = %x, want %x", fdata, data)
	}
	fsize := f.size()
	want := len(data) + size + sequence
	if int(fsize) != want {
		t.Errorf("f.size() = %d, want %d", fsize, want)
	}

	data = make([]byte, maxBytes+1)
	if _, err = newFrame(0, data); err != errFrameTooBig {
		t.Errorf("newFrame err = %v; want %v", err, errFrameTooBig)
	}

	f = make([]byte, total-1)
	if data = f.data(); len(data) != 0 {
		// Frame that is smaller than its
		// header size cannot have data.
		t.Errorf("f.data() = %x; want nil", data)
	}
	if seq := f.seq(); seq != 0 {
		// Frame that is smaller than its
		// header size cannot have data
		// and should not ret seq number.
		t.Errorf("f.seq() = %d; want nil", seq)
	}
}

func TestMaxByteValue(t *testing.T) {
	var want uint32 = 4294967284
	if maxBytes != want {
		t.Errorf("maxBytes = %d, want %d", maxBytes, want)
	}
}

func TestLargeFrame(t *testing.T) {
	mb := maxBytes
	maxBytes = 10
	defer func() { maxBytes = mb }()
	t.SkipNow() // takes a lot of memory
	data := make([]byte, maxBytes)
	if _, err := newFrame(0, data); err != nil {
		t.Errorf("Failed to create max size frame: %v", err)
	}
	data = make([]byte, maxBytes+1)
	if _, err := newFrame(0, data); err != errFrameTooBig {
		t.Errorf("Expected to fail to create frame because of size: %v", err)
	}
}

func TestNewAndLoadBuffer(t *testing.T) {
	n := 100
	b, err := New(n, bufFile)
	if err != nil {
		t.Fatalf("Failed to create Buffer: %v", err)
	}
	if !bytes.Equal(b.data[n:], make([]byte, metadata)) {
		t.Errorf("Expect 0s in metadata, got = %x", b.data[n:])
	}

	mustInsert(t, b.Insert([]byte("This is data")))
	bufData := make([]byte, len(b.data))
	copy(bufData, b.data)
	b.Unmap()

	if _, err = Load("🚫"); err == nil {
		t.Errorf("Successfully loaded buffer from non-existent file")
	}
	b, err = Load(bufFile)
	if err != nil {
		os.Remove(bufFile)
		t.Fatalf("Failed to load buf from file: %v", err)
	}
	defer b.Close()
	if !bytes.Equal(b.data, bufData) {
		t.Errorf("Load returned buf with data %x; want %x", b.data, bufData)
	}

	_, err = New(n, bufFile)
	if err != ErrFileExists {
		t.Errorf("Should fail to create new buffer if file already exists")
	}
}

func TestBufferInsert(t *testing.T) {
	n := 3 + total
	b, err := New(n, bufFile)
	if err != nil {
		t.Fatalf("Failed to create Buffer: %v", err)
	}
	defer b.Close()
	metaoff := len(b.data) - metadata

	data := []byte("123")
	if err := b.Insert(data); err != nil {
		t.Errorf("First insert failed: %v", err)
	}
	if !bytes.Equal(data, b.data[total:metaoff]) {
		t.Error("First write data encoded incorrectly")
	}

	if err := b.Insert([]byte("ab")); err != nil {
		t.Errorf("Second insert failed: %v", err)
	}
	if !bytes.Equal([]byte("ab3"), b.data[total:metaoff]) {
		t.Error("Second write data encoded incorrectly")
	}

	data = []byte("x")
	if err := b.Insert(data); err != nil {
		t.Errorf("Third insert failed: %v", err)
	}
	if !bytes.Equal(data, b.data[total-1:total]) {
		// The first byte of the frame header should have
		// been written where "3" was before wrapping.
		t.Error("Third write data encoded incorrectly")
	}
}

func TestEmptyBufferRead(t *testing.T) {
	b, err := New(10, bufFile)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer b.Close()

	var data []byte
	if _, _, err := b.ReadFirst(data); err != ErrNotArrived {
		t.Errorf("b.ReadFirst err = %v; want %v", err, ErrNotArrived)
	}
}

func TestBufferRead(t *testing.T) {
	input := [][]byte{
		[]byte("data goes in"),
		[]byte("data comes back out"),
		[]byte("salad a month"),
		[]byte("is a good thing"),
	}

	var n int
	for _, d := range input {
		n += len(d) + total
	}
	n -= 1 // slightly smaller than all data
	b, err := New(n, bufFile)
	if err != nil {
		t.Fatalf("Failed to create Buffer: %v", err)
	}
	defer b.Close()

	l := len(input)
	insertedBytes := 0
	for i := 0; i < l-1; i++ {
		data := input[i]
		insertedBytes += len(data)
		mustInsert(t, b.Insert(data))
	}

	if length := b.Len(); length != uint64(insertedBytes) {
		t.Errorf("b.Len() = %d; want %d", length, insertedBytes)
	}

	in := input[0]
	data := make([]byte, len(in))
	_, c, err := b.ReadFirst(data)
	second := c
	if err != nil {
		t.Errorf("b.ReadFirst err = %v; want nil", err)
	}
	if seq := c.Seq(); seq != 1 {
		t.Errorf("b.ReadFirst seq = %d; want 0", seq)
	}
	if !bytes.Equal(data, in) {
		t.Errorf("b.ReadFirst data = %x; want %x", data, in)
	}
	wantoff := total + len(in)
	if c.offset != uint64(wantoff) {
		t.Errorf("b.ReadFirst offset = %d; want %d", c.offset, wantoff)
	}

	// Haven't written input[3] yet.
	want := uint64(len(input[1])+len(input[2])) + 2*total
	if remaining := b.Remaining(second); remaining != want {
		t.Errorf("b.Reamining(c) = %d; want %d", remaining, want)
	}

	in = input[1]
	cutoff := len(in) - 4
	data = make([]byte, cutoff)
	_, c, err = b.Read(data, c)
	third := c
	if err != ErrPartialData {
		t.Errorf("b.Read err = %v, expect %v", err, ErrPartialData)
	}
	if !bytes.Equal(data, in[:cutoff]) {
		t.Errorf("b.Read data = %x; want %x", data, in[:cutoff])
	}

	mustInsert(t, b.Insert(input[l-1]))

	data = make([]byte, len(in))
	_, c, err = b.ReadFirst(data)
	if err != nil {
		t.Errorf("b.ReadFirst err = %v; want nil", err)
	}
	if seq := c.Seq(); seq != 2 {
		t.Errorf("b.ReadFirst seq = %d; want 2", seq)
	}
	if c.offset != third.offset {
		t.Errorf("b.Read offset = %d; want %d", c.offset, third.offset)
	}
	if !bytes.Equal(data, in) {
		t.Errorf("b.Read data = %x; want %x", data, in)
	}

	_, c, err = b.Read(data, second)
	if err != nil {
		t.Errorf("b.Read err = %v; want nil", err)
	}
	if c.offset != third.offset {
		t.Errorf("b.Read offset = %d; want %d", c.offset, third.offset)
	}

	want = uint64(len(input[1])+len(input[2])+len(input[3])) + 3*total
	if remaining := b.Remaining(second); remaining != want {
		t.Errorf("b.Reamining(c) = %d; want %d", remaining, want)
	}

	in = make([]byte, n-total)
	copy(in, []byte("Record where frame size equals capacity"))
	mustInsert(t, b.Insert(in))

	if length := b.Len(); length != uint64(len(in)) {
		t.Errorf("b.Len() = %d; want %d", length, len(in))
	}

	want = uint64(len(in)) + total
	if remaining := b.Remaining(second); remaining != want {
		t.Errorf("b.Reamining(c) = %d; want %d", remaining, want)
	}

	_, _, err = b.Read(data, second)
	if err != ErrPartialData {
		t.Errorf("b.Read err = %v; want %v", err, ErrPartialData)
	}

	data = make([]byte, len(in))
	_, c, err = b.Read(data, second)
	if err != nil {
		t.Errorf("b.Read err = %v; want nil", err)
	}
	if seq := c.Seq(); seq != 5 {
		t.Errorf("b.Read seq = %d; want 5", seq)
	}
	if !bytes.Equal(data, in) {
		t.Errorf("b.Read data = %x; want %x", data, in)
	}

	if remaining := b.Remaining(c); remaining != 0 {
		t.Errorf("b.Reamining(c) = %d; want 0", remaining)
	}

	_, _, err = b.Read(data, c)
	if err != ErrNotArrived {
		t.Errorf("b.Read err = %v; want %v", err, ErrNotArrived)
	}

	if err := b.Insert(make([]byte, n+1)); err != errDataTooBig {
		t.Errorf("b.Insert err = %v; want %v", err, errDataTooBig)
	}

	in = []byte("Adding data when the buffer can contain only one record")
	mustInsert(t, b.Insert(in))

	if length := b.Len(); length != uint64(len(in)) {
		t.Errorf("b.Len() = %d; want %d", length, len(in))
	}

	data = make([]byte, len(in))
	_, c, err = b.Read(data, c)
	if err != nil {
		t.Errorf("b.Read err = %v; want nil", err)
	}
	if seq := c.Seq(); seq != 6 {
		t.Errorf("b.Read seq = %d; want 6", seq)
	}
	if !bytes.Equal(data, in) {
		t.Errorf("b.Read data = %x; want %x", data, in)
	}
}

func TestWrappingData(t *testing.T) {
	// This test tests a special scenario (henceforth situation) where:
	//
	//   end = wrapped start + length of record
	//   b.capacity - start < end % b.capacity
	//
	// because there was a bug where partial record would not be copied
	// into the output slice.
	var (
		d1   = []byte("dragons")
		d2   = []byte("winter is coming")
		d3   = []byte("snakes are found in south of the kingdom")
		data = make([]byte, 1<<10)
	)

	if l := len(d1); l != 7 {
		t.Fatalf("len(d1) = %d; want 7 to re-created situation", l)
	}
	if l := len(d2); l != 16 {
		t.Fatalf("len(d2) = %d; want 16 to re-created situation", l)
	}
	if l := len(d3); l != 40 {
		t.Fatalf("len(d3) = %d; want 40 to re-created situation", l)
	}

	n := total + len(d3) + 1
	b, err := New(n, bufFile)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer b.Close()

	mustInsert(t, b.Insert(d1))
	mustInsert(t, b.Insert(d2))

	n, c, err := b.ReadFirst(data)
	if err != nil {
		t.Errorf("ReadFirst err = %v; want nil", err)
	}
	if n != len(d1) {
		t.Errorf("ReadFirst n = %d; want %d", n, len(d1))
	}
	got := data[:n]
	if !bytes.Equal(got, d1) {
		t.Errorf("ReadFirst got %x; want %x", got, d1)
	}

	n, c, err = b.Read(data, c)
	if err != nil {
		t.Errorf("Read err = %v; want nil", err)
	}
	if n != len(d2) {
		t.Errorf("Read n = %d; want %d", n, len(d2))
	}
	got = data[:n]
	if !bytes.Equal(got, d2) {
		t.Errorf("Read got %x; want %x", got, d2)
	}

	_, _, err = b.Read(data, c)
	if err != ErrNotArrived {
		t.Errorf("Read err = %v; want %v", err, ErrNotArrived)
	}

	mustInsert(t, b.Insert(d3))

	n, c, err = b.Read(data, c)
	if err != nil {
		t.Errorf("Read err = %v; want nil", err)
	}
	if n != len(d3) {
		t.Errorf("Read n = %d; want %d", n, len(d3))
	}
	got = data[:n]
	if !bytes.Equal(got, d3) {
		t.Errorf("Read got %x; want %x", got, d3)
	}
}

func TestOffByOneWrapping(t *testing.T) {
	var (
		s = 10
		n = 2 * (s + total)
		d = make([]byte, s)
	)

	for i := 0; i < len(d); i++ {
		d[i] = 42
	}

	b, err := New(n, bufFile)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer b.Close()

	mustInsert(t, b.Insert(d))
	mustInsert(t, b.Insert(d))
	mustInsert(t, b.Insert(d))
	mustInsert(t, b.Insert(d))
	if l := b.Len(); l != 20 {
		// Using length as a proxy to test wrapping
		// is done right when the record ejected
		// after previous write ends just at the
		// last byte.
		t.Errorf("b.Len() = %d; want 20", l)
	}
}

func TestAdviseOne(t *testing.T) {
	n := os.Getpagesize()
	b, err := New(n+total, bufFile)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer b.Close()

	data := make([]byte, n)
	mustInsert(t, b.Insert(data))
	_, c, err := b.ReadFirst(data)
	if err != nil {
		t.Fatalf("b.ReadFirst err = %v; want nil", err)
	}
	if err = b.DontNeed(c); err != nil {
		t.Errorf("b.DontNeed = %v, want nil", err)
	}
}

func TestAdviseTwo(t *testing.T) {
	p := os.Getpagesize()
	b, err := New(2*p, bufFile)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer b.Close()

	mustInsert(t, b.Insert(make([]byte, p-total)))
	data := make([]byte, 2*p-total)
	mustInsert(t, b.Insert(data))

	_, c, err := b.ReadFirst(data)
	if err != nil {
		t.Fatalf("b.ReadFirst err = %v; want nil", err)
	}
	if err = b.DontNeed(c); err != nil {
		t.Errorf("b.DontNeed = %v, want nil", err)
	}
}

func mustInsert(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
}

func BenchmarkRead(b *testing.B) {
	var input [][]byte
	isize := 1024
	for i := 0; i < 50; i++ {
		data := make([]byte, isize)
		io.ReadFull(rand.Reader, data)
		input = append(input, data)
	}
	f, err := ioutil.TempFile("", "buffer-bench-")
	if err != nil {
		b.Fatalf("Cannot create temp file: %v", err)
	}
	name := f.Name()
	if err := os.Remove(name); err != nil {
		b.Fatalf("Cannot remove temp file: %v", err)
	}

	buf, err := New(10<<20, f.Name())
	if err != nil {
		b.Fatalf("Failed to created buf: %v", err)
	}
	defer buf.Close()

	b.ResetTimer()

	closec := make(chan struct{})
	go insertAll(buf, input, closec)

	data := make([]byte, isize)
	_, c, err := buf.ReadFirst(data)
	if err != nil && err != ErrPartialData && err != ErrNotArrived {
		b.Fatalf("buf.ReadFirst err = %v", err)
	}
	for i := 0; i < b.N; i++ {
		_, c, err = buf.Read(data, c)
		if err != nil && err != ErrPartialData && err != ErrNotArrived {
			b.Fatalf("buf.Read err = %v", err)
		}
	}
	close(closec)
}

func insertAll(b *Buffer, input [][]byte, closec <-chan struct{}) {
	inc := make(chan []byte)
	for i := 0; i < 12; i++ {
		go inserter(b, inc)
	}
	l := len(input)
	for i := 0; ; i++ {
		select {
		case <-closec:
			close(inc)
			return
		default:
			inc <- input[i%l]
		}
	}
}

func inserter(b *Buffer, inc <-chan []byte) {
	for data := range inc {
		b.Insert(data)
	}
}
