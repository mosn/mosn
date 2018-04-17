package buffer

import (
	"testing"
)

func Test_read(t *testing.T) {
	str := "read_test"
	buffer := NewIoBufferString(str)

	b := make([]byte, 32)
	_, err := buffer.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if string(b[:len(str)]) != str {
		t.Fatal("err read content")
	}

	buffer = NewIoBufferString(str)

	b = make([]byte, 4)
	_, err = buffer.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if string(b) != "read" {
		t.Fatal("err read content")
	}
}
