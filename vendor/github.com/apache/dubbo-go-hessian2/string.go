/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package hessian

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"unicode/utf8"
	"unsafe"
)

import (
	gxbytes "github.com/dubbogo/gost/bytes"
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

// NOTE: The length of hessian string is the number of 16-bit characters,
// which may be different than the number of bytes.
// String chunks may not split surrogate pairs.
//
// While golang support ucs-4, a rune may exceed 16-bit, which need convert to ucs-2.
//
// ref:
// - https://en.wikipedia.org/wiki/UTF-16
// - https://en.wikipedia.org/wiki/UCS-4
// - http://www.unicode.org/glossary/#code_point
func encodeUcs4Rune(b []byte, r rune) (int, int) {
	if r >= 0x10000 && r <= 0x10FFFF {
		t := uint32(r) - 0x10000
		n := encodeUcs2Rune(b, t>>10+0xD800)
		n += encodeUcs2Rune(b[n:], t&0x3FF+0xDC00)
		return n, 2
	}

	// in fact, a rune over 0x10FFFF can't be encoded by hessian, ignore it currently
	return utf8.EncodeRune(b, r), 1
}

func encodeUcs2Rune(b []byte, ch uint32) int {
	if ch < 0x80 {
		b[0] = byte(ch)
		return 1
	}

	if ch < 0x800 {
		b[0] = byte(0xc0 + ((ch >> 6) & 0x1f))
		b[1] = byte(0x80 + (ch & 0x3f))
		return 2
	}

	b[0] = byte(0xe0 + ((ch >> 12) & 0x0f))
	b[1] = byte(0x80 + ((ch >> 6) & 0x3f))
	b[2] = byte(0x80 + (ch & 0x3f))

	return 3
}

// # UTF-8 encoded character string split into 64k chunks
// ::= x52 b1 b0 <utf8-data> string  # non-final chunk
// ::= 'S' b1 b0 <utf8-data>         # string of length 0-65535
// ::= [x00-x1f] <utf8-data>         # string of length 0-31
// ::= [x30-x34] <utf8-data>         # string of length 0-1023
func encString(b []byte, v string) []byte {
	if v == "" {
		return encByte(b, BC_STRING_DIRECT)
	}

	var (
		byteLen int
		charLen int
		vBuf    = *bytes.NewBufferString(v)

		byteRead  int
		charCount int
		byteCount int
	)

	// Acquire (CHUNK_SIZE + 1) * 3 bytes since charCount could reach CHUNK_SIZE + 1.
	bufp := gxbytes.AcquireBytes((CHUNK_SIZE + 1) * 3)
	defer gxbytes.ReleaseBytes(bufp)
	buf := *bufp

	byteRead = 0
	for {
		if vBuf.Len() <= 0 {
			break
		}

		charCount = 0
		byteCount = 0
		for charCount < CHUNK_SIZE {
			r, _, err := vBuf.ReadRune()
			if err != nil {
				break
			}

			byteLen, charLen = encodeUcs4Rune(buf[byteCount:], r)
			charCount += charLen
			byteCount += byteLen
		}

		if charCount == 0 {
			break
		}

		switch {
		case vBuf.Len() > 0 && charCount >= CHUNK_SIZE:
			b = encByte(b, BC_STRING_CHUNK)
			b = encByte(b, PackUint16(uint16(charCount))...)
		case charCount <= int(STRING_DIRECT_MAX):
			b = encByte(b, byte(charCount+int(BC_STRING_DIRECT)))
		case charCount <= STRING_SHORT_MAX:
			b = encByte(b, byte((charCount>>8)+int(BC_STRING_SHORT)), byte(charCount))
		default:
			b = encByte(b, BC_STRING)
			b = encByte(b, PackUint16(uint16(charCount))...)
		}

		b = append(b, buf[:byteCount]...)

		byteRead = byteRead + byteCount
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
func (d *Decoder) getStringLength(tag byte) (int, error) {
	var length int

	switch {
	case tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX:
		return int(tag - 0x00), nil

	case tag >= 0x30 && tag <= 0x33:
		b, err := d.ReadByte()
		if err != nil {
			return -1, perrors.WithStack(err)
		}

		length = int(tag-0x30)<<8 + int(b)
		return length, nil

	case tag == BC_STRING_CHUNK || tag == BC_STRING:
		b0, err := d.ReadByte()
		if err != nil {
			return -1, perrors.WithStack(err)
		}

		b1, err := d.ReadByte()
		if err != nil {
			return -1, perrors.WithStack(err)
		}

		length = int(b0)<<8 + int(b1)
		return length, nil

	default:
		return -1, perrors.Errorf("string decode: unknown tag %b", tag)
	}
}

func (d *Decoder) decString(flag int32) (string, error) {
	var (
		tag byte
		s   string
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.ReadByte()
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

		if tag != BC_STRING_CHUNK {
			data, err := d.readStringChunkData(tag)
			if err != nil {
				return "", err
			}
			return *(*string)(unsafe.Pointer(&data)), nil
		}

		var chunkDataSlice [][]byte
		dataLength := 0

		for {
			data, err := d.readStringChunkData(tag)
			if err != nil {
				return "", err
			}

			chunkDataSlice = append(chunkDataSlice, data)
			dataLength += len(data)

			// last chunk
			if tag != BC_STRING_CHUNK {
				allData := make([]byte, dataLength)
				index := 0
				for _, b := range chunkDataSlice {
					copy(allData[index:], b)
					index += len(b)
				}
				return *(*string)(unsafe.Pointer(&allData)), nil
			}

			// read next string chunk tag
			tag, _ = d.ReadByte()
			switch {
			case (tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX) ||
				(tag >= 0x30 && tag <= 0x33) ||
				(tag == BC_STRING_CHUNK || tag == BC_STRING):

			default:
				return s, perrors.New("expect string tag")
			}
		}

	}

	return s, perrors.Errorf("unknown string tag %#x\n", tag)
}

// readStringChunkData read one string chunk data as a utf8 buffer
func (d *Decoder) readStringChunkData(tag byte) ([]byte, error) {
	charTotal, err := d.getStringLength(tag)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	data := make([]byte, charTotal*3)

	start := 0
	end := 0

	charCount := 0
	charRead := 0

	for charCount < charTotal {
		_, err = io.ReadFull(d.reader, data[end:end+charTotal-charCount])
		if err != nil {
			return nil, err
		}

		end += charTotal - charCount

		start, end, charRead, err = decode2utf8(d.reader, data, start, end)
		if err != nil {
			return nil, err
		}

		charCount += charRead
	}

	return data[:end], nil
}

// decode2utf8 decode hessian2 buffer to utf8 buffer
// parameters:
// - r : the input buffer
// - data: the buffer already read
// - start: the decoding index
// - end: the already read buffer index
// response: updated start, updated end, read char count, error.
func decode2utf8(r *bufio.Reader, data []byte, start, end int) (int, int, int, error) {
	var err error

	charCount := 0

	for start < end {
		ch := data[start]
		if ch < 0x80 {
			start++
			charCount++
			continue
		}

		if start+1 == end {
			data[end], err = r.ReadByte()
			if err != nil {
				return start, end, 0, err
			}
			end++
		}

		if (ch & 0xe0) == 0xc0 {
			start += 2
			charCount++
			continue
		}

		if start+2 == end {
			data[end], err = r.ReadByte()
			if err != nil {
				return start, end, 0, err
			}
			end++
		}

		if (ch & 0xf0) == 0xe0 {
			c1 := ((uint32(ch) & 0x0f) << 12) + ((uint32(data[start+1]) & 0x3f) << 6) + (uint32(data[start+2]) & 0x3f)

			if c1 >= 0xD800 && c1 <= 0xDBFF {
				if start+6 >= end {
					_, err = io.ReadFull(r, data[end:start+6])
					if err != nil {
						return start, end, 0, err
					}
					end = start + 6
				}

				c2 := ((uint32(data[start+3]) & 0x0f) << 12) + ((uint32(data[start+4]) & 0x3f) << 6) + (uint32(data[start+5]) & 0x3f)
				c := (c1-0xD800)<<10 + (c2 - 0xDC00) + 0x10000

				n := utf8.EncodeRune(data[start:], rune(c))
				copy(data[start+n:], data[start+6:end])
				start, end = start+n, end-6+n

				charCount += 2
				continue
			}

			start += 3
			charCount++
			continue
		}

		return start, end, 0, fmt.Errorf("bad utf-8 encoding at %x", ch)
	}

	return start, end, charCount, nil
}
