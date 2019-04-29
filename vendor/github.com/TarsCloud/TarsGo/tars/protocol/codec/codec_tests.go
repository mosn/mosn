package codec

import (
	"math"
	"testing"
)

func r(b *Buffer) *Reader {
	buf := b.ToBytes()
	return NewReader(buf)
}

func TestUint8(t *testing.T) {
	for tag := 0; tag < 250; tag++ {
		for i := 0; i <= math.MaxUint8; i++ {
			b := NewBuffer()
			err := b.Write_uint8(uint8(i), byte(tag))
			if err != nil {
				t.Error(err)
			}
			var data uint8
			err = r(b).Read_uint8(&data, byte(tag), true)
			if err != nil {
				t.Error(err)
			}
			if data != uint8(i) {
				t.Error("no eq.")
			}
		}
	}
}

func TestInt8(t *testing.T) {
	for tag := 0; tag < 250; tag++ {
		for i := math.MinInt8; i <= math.MaxInt8; i++ {
			b := NewBuffer()
			err := b.Write_int8(int8(i), byte(tag))
			if err != nil {
				t.Error(err)
			}
			var data int8
			err = r(b).Read_int8(&data, byte(tag), true)
			if err != nil {
				t.Error(err)
			}
			if data != int8(i) {
				t.Error("no eq.")
			}
		}
	}
}

func TestUint16(t *testing.T) {
	for tag := 0; tag < 250; tag += 10 {
		for i := 0; i < math.MaxUint16; i++ {
			b := NewBuffer()
			err := b.Write_uint16(uint16(i), byte(tag))
			if err != nil {
				t.Error(err)
			}
			var data uint16
			err = r(b).Read_uint16(&data, byte(tag), true)
			if err != nil {
				t.Error(err)
			}
			if data != uint16(i) {
				t.Error("no eq.")
			}
		}
	}
}

func TestInt16(t *testing.T) {
	for tag := 0; tag < 250; tag += 10 {
		for i := math.MinInt16; i <= math.MaxInt16; i++ {
			b := NewBuffer()
			err := b.Write_int16(int16(i), byte(tag))
			if err != nil {
				t.Error(err)
			}
			var data int16
			err = r(b).Read_int16(&data, byte(tag), true)
			if err != nil {
				t.Error(err)
			}
			if data != int16(i) {
				t.Error("no eq.", i)
			}
		}
	}
}

func TestInt16_2(t *testing.T) {
	b := NewBuffer()
	err := b.Write_int16(int16(-1), byte(0))
	if err != nil {
		t.Error(err)
	}
	var data int16
	err = r(b).Read_int16(&data, byte(0), true)
	if err != nil {
		t.Error(err)
	}
	if data != int16(-1) {
		t.Error("no eq.", data)
	}
}

func TestInt32(t *testing.T) {
	b := NewBuffer()
	err := b.Write_int32(int32(-1), byte(10))
	if err != nil {
		t.Error(err)
	}
	var data int32
	err = r(b).Read_int32(&data, byte(10), true)
	if err != nil {
		t.Error(err)
	}
	if data != -1 {
		t.Error("no eq.")
	}
}

func TestInt32_2(t *testing.T) {
	b := NewBuffer()
	err := b.Write_int32(math.MinInt32, byte(10))
	if err != nil {
		t.Error(err)
	}
	var data int32
	err = r(b).Read_int32(&data, byte(10), true)
	if err != nil {
		t.Error(err)
	}
	if data != math.MinInt32 {
		t.Error("no eq.")
	}
}
func TestUint32(t *testing.T) {
	b := NewBuffer()
	err := b.Write_uint32(uint32(0xffffffff), byte(10))
	if err != nil {
		t.Error(err)
	}
	var data uint32
	err = r(b).Read_uint32(&data, byte(10), true)
	if err != nil {
		t.Error(err)
	}
	if data != 0xffffffff {
		t.Error("no eq.")
	}
}

func TestInt64(t *testing.T) {
	b := NewBuffer()
	err := b.Write_int64(math.MinInt64, byte(10))
	if err != nil {
		t.Error(err)
	}
	var data int64
	err = r(b).Read_int64(&data, byte(10), true)
	if err != nil {
		t.Error(err)
	}
	if data != math.MinInt64 {
		t.Error("no eq.")
	}
}

func TestSkipString(t *testing.T) {
	b := NewBuffer()
	for i := 0; i < 200; i++ {
		bs := make([]byte, 200+i)
		err := b.Write_string(string(bs), byte(i))
		if err != nil {
			t.Error(err)
		}
	}

	var data string
	err := r(b).Read_string(&data, byte(190), true)
	if err != nil {
		t.Error(err)
	}
	bs := make([]byte, 200+190)
	if data != string(bs) {
		t.Error("no eq.")
	}
}

func TestSkipStruct(t *testing.T) {
	b := NewBuffer()

	err := b.WriteHead(STRUCT_BEGIN, 1)
	if err != nil {
		t.Error(err)
	}

	err = b.WriteHead(STRUCT_END, 0)
	if err != nil {
		t.Error(err)
	}

	rd := r(b)

	err, have := rd.SkipTo(STRUCT_BEGIN, 1, true)
	if err != nil || have == false {
		t.Error(err)
	}
	err = rd.SkipToStructEnd()
	if err != nil || have == false {
		t.Error(err)
	}
}

func TestSkipStruct2(t *testing.T) {
	b := NewBuffer()

	err := b.WriteHead(STRUCT_BEGIN, 1)
	if err != nil {
		t.Error(err)
	}
	err = b.WriteHead(STRUCT_BEGIN, 1)
	if err != nil {
		t.Error(err)
	}

	err = b.WriteHead(STRUCT_END, 0)
	if err != nil {
		t.Error(err)
	}
	err = b.WriteHead(STRUCT_END, 0)
	if err != nil {
		t.Error(err)
	}
	err = b.Write_int64(math.MinInt64, byte(10))
	if err != nil {
		t.Error(err)
	}

	rb := r(b)

	err, have := rb.SkipTo(STRUCT_BEGIN, 1, true)
	if err != nil || !have {
		t.Error(err)
	}
	err = rb.SkipToStructEnd()
	if err != nil {
		t.Error(err)
	}
	var data int64
	err = rb.Read_int64(&data, byte(10), true)
	if err != nil {
		t.Error(err)
	}
	if data != math.MinInt64 {
		t.Error("no eq.")
	}
}

func BenchmarkUint32(t *testing.B) {
	b := NewBuffer()

	for i := 0; i < 200; i++ {
		err := b.Write_uint32(uint32(0xffffffff), byte(i))
		if err != nil {
			t.Error(err)
		}
	}

	rb := r(b)

	for i := 0; i < 200; i++ {
		var data uint32
		err := rb.Read_uint32(&data, byte(i), true)
		if err != nil {
			t.Error(err)
		}
		if data != 0xffffffff {
			t.Error("no eq.")
		}
	}
}

func BenchmarkString(t *testing.B) {
	b := NewBuffer()

	for i := 0; i < 200; i++ {
		err := b.Write_string("hahahahahahahahahahahahahahahahahahahaha", byte(i))
		if err != nil {
			t.Error(err)
		}
	}

	rb := r(b)

	for i := 0; i < 200; i++ {
		var data string
		err := rb.Read_string(&data, byte(i), true)
		if err != nil {
			t.Error(err)
		}
		if data != "hahahahahahahahahahahahahahahahahahahaha" {
			t.Error("no eq.")
		}
	}
}
