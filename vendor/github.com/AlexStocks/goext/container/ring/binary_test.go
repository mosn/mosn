package gxring

import (
	"bytes"
	"testing"
)

func TestLittleEndian(t *testing.T) {
	buffer := make([]byte, 14)
	PutLittleEndianUint32(buffer, 2, 0x06050403)
	PutLittleEndianUint64(buffer, 6, 0x0e0d0c0b0a090807)

	want := []byte{0x00, 0x00, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e}
	if !bytes.Equal(buffer, want) {
		t.Fatal("buffer = %x; want = %x", buffer, want)
	}

	x := GetLittleEndianUint32(buffer, 2)
	if want := uint32(0x06050403); x != want {
		t.Fatalf("GetLittleEndianUint32 = %d; want = %d", x, want)
	}

	y := GetLittleEndianUint64(buffer, 6)
	if want := uint64(0x0e0d0c0b0a090807); y != want {
		t.Fatalf("GetLittleEndianUint64 = %d; want = %d", x, want)
		t.Fatal("64")
	}
}
