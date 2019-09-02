// Copyright 2016-2019 Alex Stocks
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hessian

import (
	"bytes"
	"io"
	"reflect"
	"strconv"
	"unicode/utf8"
	"unsafe"
)

import (
	perrors "github.com/pkg/errors"
)

/////////////////////////////////////////
// String
/////////////////////////////////////////

// Slice convert string to byte slice
func Slice(s string) (b []byte) {
	pbytes := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	pstring := (*reflect.StringHeader)(unsafe.Pointer(&s))
	pbytes.Data = pstring.Data
	pbytes.Len = pstring.Len
	pbytes.Cap = pstring.Len
	return
}

// # UTF-8 encoded character string split into 64k chunks
// ::= x52 b1 b0 <utf8-data> string  # non-final chunk
// ::= 'S' b1 b0 <utf8-data>         # string of length 0-65535
// ::= [x00-x1f] <utf8-data>         # string of length 0-31
// ::= [x30-x34] <utf8-data>         # string of length 0-1023
func encString(b []byte, v string) []byte {
	var (
		vLen int

		vBuf = *bytes.NewBufferString(v)

		vChunk = func(length int) {
			for i := 0; i < length; i++ {
				if r, s, err := vBuf.ReadRune(); s > 0 && err == nil {
					// b = append(b, []byte(string(r))...)
					b = append(b, Slice(string(r))...) // converts it to []byte in memory space of "r"
				}
			}
		}
	)

	if v == "" {
		return encByte(b, BC_STRING_DIRECT)
	}

	for {
		vLen = utf8.RuneCount(vBuf.Bytes())
		if vLen == 0 {
			break
		}
		if vLen > CHUNK_SIZE {
			b = encByte(b, BC_STRING_CHUNK)
			b = encByte(b, PackUint16(uint16(CHUNK_SIZE))...)
			vChunk(CHUNK_SIZE)
		} else {
			if vLen <= int(STRING_DIRECT_MAX) {
				b = encByte(b, byte(vLen+int(BC_STRING_DIRECT)))
			} else if vLen <= int(STRING_SHORT_MAX) {
				b = encByte(b, byte((vLen>>8)+int(BC_STRING_SHORT)), byte(vLen))
			} else {
				b = encByte(b, BC_STRING)
				b = encByte(b, PackUint16(uint16(vLen))...)
			}
			vChunk(vLen)
		}
	}

	return b
}

/////////////////////////////////////////
// String
/////////////////////////////////////////

// # UTF-8 encoded character string split into 64k chunks
// ::= x52 b1 b0 <utf8-data> string  # non-final chunk
// ::= 'S' b1 b0 <utf8-data>         # string of length 0-65535
// ::= [x00-x1f] <utf8-data>         # string of length 0-31
// ::= [x30-x34] <utf8-data>         # string of length 0-1023
func (d *Decoder) getStringLength(tag byte) (int32, error) {
	var (
		err    error
		buf    [2]byte
		length int32
	)

	switch {
	case tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX:
		return int32(tag - 0x00), nil

	case tag >= 0x30 && tag <= 0x33:
		_, err = io.ReadFull(d.reader, buf[:1])
		if err != nil {
			return -1, perrors.WithStack(err)
		}

		length = int32(tag-0x30)<<8 + int32(buf[0])
		return length, nil

	case tag == BC_STRING_CHUNK || tag == BC_STRING:
		_, err = io.ReadFull(d.reader, buf[:2])
		if err != nil {
			return -1, perrors.WithStack(err)
		}
		length = int32(buf[0])<<8 + int32(buf[1])
		return length, nil

	default:
		return -1, perrors.WithStack(err)
	}
}

// hessian-lite/src/main/java/com/alibaba/com/caucho/hessian/io/Hessian2Input.java : readString
func (d *Decoder) decString(flag int32) (string, error) {
	var (
		tag    byte
		length int32
		last   bool
		s      string
		r      rune
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}

	switch {
	case tag == BC_NULL:
		return STRING_NIL, nil

	case tag == BC_TRUE:
		return STRING_TRUE, nil

	case tag == BC_FALSE:
		return STRING_FALSE, nil

	case (0x80 <= tag && tag <= 0xbf) || (0xc0 <= tag && tag <= 0xcf) ||
		(0xd0 <= tag && tag <= 0xd7) || tag == BC_INT ||
		(tag >= 0xd8 && tag <= 0xef) || (tag >= 0xf0 && tag <= 0xff) ||
		(tag >= 0x38 && tag <= 0x3f) || (tag == BC_LONG_INT) || (tag == BC_LONG):
		i64, err := d.decInt64(int32(tag))
		if err != nil {
			return "", perrors.Wrapf(err, "tag:%+v", tag)
		}

		return strconv.Itoa(int(i64)), nil

	case tag == BC_DOUBLE_ZERO:
		return STRING_ZERO, nil

	case tag == BC_DOUBLE_ONE:
		return STRING_ONE, nil

	case tag == BC_DOUBLE_BYTE || tag == BC_DOUBLE_SHORT:
		f, err := d.decDouble(int32(tag))
		if err != nil {
			return "", perrors.Wrapf(err, "tag:%+v", tag)
		}

		return strconv.FormatFloat(f.(float64), 'E', -1, 64), nil
	}

	if (tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX) ||
		(tag >= 0x30 && tag <= 0x33) ||
		(tag == BC_STRING_CHUNK || tag == BC_STRING) {

		if tag == BC_STRING_CHUNK {
			last = false
		} else {
			last = true
		}

		l, err := d.getStringLength(tag)
		if err != nil {
			return s, perrors.WithStack(err)
		}
		length = l
		runeDate := make([]rune, length)
		for i := 0; ; {
			if int32(i) == length {
				if last {
					return string(runeDate), nil
				}

				b, _ := d.readByte()
				switch {
				case (tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX) ||
					(tag >= 0x30 && tag <= 0x33) ||
					(tag == BC_STRING_CHUNK || tag == BC_STRING):

					if b == BC_STRING_CHUNK {
						last = false
					} else {
						last = true
					}

					l, err := d.getStringLength(b)
					if err != nil {
						return s, perrors.WithStack(err)
					}
					length += l
					bs := make([]rune, length)
					copy(bs, runeDate)
					runeDate = bs

				default:
					return s, perrors.WithStack(err)
				}

			} else {
				r, _, err = d.reader.ReadRune()
				if err != nil {
					return s, perrors.WithStack(err)
				}
				runeDate[i] = r
				i++
			}
		}

		// unreachable
		// return string(runeDate), nil
	}

	return s, perrors.Errorf("unknown string tag %#x\n", tag)
}
