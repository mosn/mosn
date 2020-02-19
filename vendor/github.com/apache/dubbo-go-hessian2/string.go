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

func decodeUcs4Rune(r *bufio.Reader) (c rune, cLen, bLen int, err error) {
	c1, n1, err1 := decodeUcs2Rune(r)
	if err1 != nil {
		return c1, 0, n1, err1
	}

	if c1 >= 0xD800 && c1 <= 0xDBFF {
		c2, n2, err2 := decodeUcs2Rune(r)
		if err2 != nil {
			return c2, 0, n2, err2
		}

		c := (c1-0xD800)<<10 + (c2 - 0xDC00) + 0x10000
		return c, 2, n1 + n2, nil
	}

	return c1, 1, n1, nil
}

func decodeUcs2Rune(r *bufio.Reader) (rune, int, error) {
	ch, err := r.ReadByte()
	if err != nil {
		return utf8.RuneError, 1, err
	}

	if ch < 0x80 {
		return rune(ch), 1, nil
	}

	if (ch & 0xe0) == 0xc0 {
		ch1, err := r.ReadByte()
		if err != nil {
			return utf8.RuneError, 2, err
		}
		return rune(((uint32(ch) & 0x1f) << 6) + (uint32(ch1) & 0x3f)), 2, nil
	}

	if (ch & 0xf0) == 0xe0 {
		ch1, err := r.ReadByte()
		if err != nil {
			return utf8.RuneError, 2, err
		}
		ch2, err := r.ReadByte()
		if err != nil {
			return utf8.RuneError, 3, err
		}
		c := ((uint32(ch) & 0x0f) << 12) + ((uint32(ch1) & 0x3f) << 6) + (uint32(ch2) & 0x3f)
		return rune(c), 3, nil
	}

	return utf8.RuneError, 0, fmt.Errorf("bad utf-8 encoding at %x", ch)
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
		byteLen = 0
		charLen = 0
		vBuf    = *bytes.NewBufferString(v)

		byteRead  = 0
		charCount = 0
		byteCount = 0
	)

	bufp := gxbytes.GetBytes(CHUNK_SIZE * 3)
	buf := *bufp
	defer gxbytes.PutBytes(bufp)

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

func (d *Decoder) decString(flag int32) (string, error) {
	var (
		tag       byte
		charTotal int32
		last      bool
		s         string
		r         rune
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
		charTotal = l
		charCount := 0

		runeData := make([]rune, charTotal)
		runeIndex := 0

		byteCount := 0
		byteLen := 0
		charLen := 0

		for {
			if int32(charCount) == charTotal {
				if last {
					return string(runeData[:runeIndex]), nil
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
					charTotal += l
					bs := make([]rune, charTotal)
					copy(bs, runeData)
					runeData = bs

				default:
					return s, perrors.New("expect string tag")
				}
			}

			r, charLen, byteLen, err = decodeUcs4Rune(d.reader)
			if err != nil {
				if err == io.EOF {
					break
				}
				return s, perrors.WithStack(err)
			}

			runeData[runeIndex] = r
			runeIndex++

			charCount += charLen
			byteCount += byteLen
		}

		return string(runeData[:runeIndex]), nil
	}

	return s, perrors.Errorf("unknown string tag %#x\n", tag)
}
