package buffer

import (
	"testing"
)

func TestIoBufferPoolWithCount(t *testing.T) {
	buf := GetIoBuffer(0)
	bytes := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	buf.Write(bytes)
	if buf.Len() != len(bytes) {
		t.Error("iobuffer len not match write bytes' size")
	}
	// Add a count, need put twice to free buffer
	buf.Count(1)
	PutIoBuffer(buf)
	if buf.Len() != len(bytes) {
		t.Error("iobuffer expected put ignore")
	}
	PutIoBuffer(buf)
	if buf.Len() != 0 {
		t.Error("iobuffer expected put success")
	}

}
