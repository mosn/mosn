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

	bufp := gxbytes.AcquireBytes(CHUNK_SIZE * 3)
	defer gxbytes.ReleaseBytes(bufp)
	buf := *bufp

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
		b, err := d.readByte()
		if err != nil {
			return -1, perrors.WithStack(err)
		}

		length = int(tag-0x30)<<8 + int(b)
		return length, nil

	case tag == BC_STRING_CHUNK || tag == BC_STRING:
		b0, err := d.readByte()
		if err != nil {
			return -1, perrors.WithStack(err)
		}

		b1, err := d.readByte()
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
		tag  byte
		last bool
		s    string
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

		chunkLen, err := d.getStringLength(tag)
		if err != nil {
			return s, perrors.WithStack(err)
		}
		bytesBuf := make([]byte, chunkLen<<2)
		offset := 0

		for {
			if chunkLen <= 0 {
				if last {
					b := bytesBuf[:offset]
					return *(*string)(unsafe.Pointer(&b)), nil
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

					chunkLen, err = d.getStringLength(b)
					if err != nil {
						return s, perrors.WithStack(err)
					}
					remain, cap := len(bytesBuf)-offset, chunkLen<<2
					if remain < cap {
						grow := len(bytesBuf) + cap
						bs := make([]byte, grow)
						copy(bs, bytesBuf)
						bytesBuf = bs
					}
				default:
					return s, perrors.New("expect string tag")
				}
			}

			if chunkLen > 0 {
				nread, err := d.next(bytesBuf[offset : offset+chunkLen])
				if err != nil {
					if err == io.EOF {
						break
					}
					return s, perrors.WithStack(err)
				}

				// quickly detect the actual number of bytes
				prev, i := offset, offset
				len := offset + nread
				copied := false
				for r, r1 := len-1, len-2; i < len; chunkLen-- {
					ch := bytesBuf[offset]
					if ch < 0x80 {
						i++
						offset++
					} else if (ch & 0xe0) == 0xc0 {
						i += 2
						offset += 2
					} else if (ch & 0xf0) == 0xe0 {
						// handle the 3-byte right edge
						// case:
						// 1. Expect 3 bytes, but the current byte is on the right
						// 2. Expect 3 bytes, but the current byte is second to last to the right
						if i == r {
							bytesBuf[i+1], err = d.reader.ReadByte()
							if err != nil {
								return s, perrors.WithStack(err)
							}
							bytesBuf[i+2], err = d.reader.ReadByte()
							if err != nil {
								return s, perrors.WithStack(err)
							}
							nread += 2
							len += 2
						} else if i == r1 {
							bytesBuf[i+2], err = d.reader.ReadByte()
							if err != nil {
								return s, perrors.WithStack(err)
							}
							nread++
							len++
						}

						// we detect emoji first
						c1 := ((uint32(ch) & 0x0f) << 12) + ((uint32(bytesBuf[i+1]) & 0x3f) << 6) + (uint32(bytesBuf[i+2]) & 0x3f)
						if c1 >= 0xD800 && c1 <= 0xDBFF {

							var (
								c2  rune
								n2  int
								err error
								ch0 byte
							)

							// more cache byte available
							if i+3 < len {
								ch0 = bytesBuf[i+3]
							} else {
								ch0, err = d.reader.ReadByte()
								if err != nil {
									return s, perrors.WithStack(err)
								}
								// update accumulates read bytes,
								// because it reads more than thunk bytes
								nread++
								len++
							}

							if ch0 < 0x80 {
								c2, n2 = rune(ch0), 1
							} else if (ch0 & 0xe0) == 0xc0 {
								var ch1 byte
								if i+4 < len {
									ch1 = bytesBuf[i+4]
								} else {
									// out of the chunk byte data
									bytesBuf[i+4], err = d.reader.ReadByte()
									ch1 = bytesBuf[i+4]
									nread++
									len++
								}
								c2, n2 = rune(((uint32(ch0)&0x1f)<<6)+(uint32(ch1)&0x3f)), 2
							} else if (ch0 & 0xf0) == 0xe0 {
								var ch1, ch2 byte
								if i+5 < len {
									ch1 = bytesBuf[i+4]
									ch2 = bytesBuf[i+5]
								} else {
									ch1, err = d.reader.ReadByte()
									if err != nil {
										return s, perrors.WithStack(err)
									}
									ch2, err = d.reader.ReadByte()
									len += 2
									nread += 2
								}
								c := ((uint32(ch0) & 0x0f) << 12) + ((uint32(ch1) & 0x3f) << 6) + (uint32(ch2) & 0x3f)
								c2, n2 = rune(c), 3
							}

							c := rune(c1-0xD800)<<10 + (c2 - 0xDC00) + 0x10000
							n3 := utf8.EncodeRune(bytesBuf[i:], c)
							if copied = n3 > 0 && n3 < /** front three byte */ 3+n2; copied {
								// We need to move the bytes,
								// for example, less bytes after decoding
								offset = i + n3
								copy(bytesBuf[offset:], bytesBuf[i+3+n2:len])
							}

							i += n2
							chunkLen--
						}
						i += 3

						// fix read the next byte index
						if copied {
							copied = false
							continue
						}

						offset += 3
					} else {
						return s, perrors.Errorf("bad utf-8 encoding")
					}
				}

				if remain := offset - prev - nread; remain > 0 {
					if remain == 1 {
						ch, err := d.readByte()
						if err != nil {
							return s, perrors.WithStack(err)
						}
						bytesBuf[offset-1] = ch
					} else {
						var err error
						if buffed := d.Buffered(); buffed < remain {
							// trigger fill data if required
							copy(bytesBuf[offset-remain:offset], d.peek(remain))
							_, err = d.reader.Discard(remain)
						} else {
							// copy remaining bytes.
							_, err = d.next(bytesBuf[offset-remain : offset])
						}

						if err != nil {
							return s, perrors.WithStack(err)
						}
					}
				}

				// the expected length string has been processed.
				if chunkLen <= 0 {
					// we need to detect next chunk
					continue
				}
			}

			// decode byte
			ch, err := d.readByte()
			if err != nil {
				if err == io.EOF {
					break
				}
				return s, perrors.WithStack(err)
			}

			if ch < 0x80 {
				bytesBuf[offset] = ch
				offset++
			} else if (ch & 0xe0) == 0xc0 {
				ch1, err := d.readByte()
				if err != nil {
					return s, perrors.WithStack(err)
				}
				bytesBuf[offset] = ch
				bytesBuf[offset+1] = ch1
				offset += 2
			} else if (ch & 0xf0) == 0xe0 {
				var err error
				if buffed := d.Buffered(); buffed < 2 {
					// trigger fill data if required
					copy(bytesBuf[offset+1:offset+3], d.peek(2))
					_, err = d.reader.Discard(2)
				} else {
					_, err = d.next(bytesBuf[offset+1 : offset+3])
				}
				if err != nil {
					return s, perrors.WithStack(err)
				}

				bytesBuf[offset] = ch

				// we detect emoji first
				c1 := ((uint32(ch) & 0x0f) << 12) + ((uint32(bytesBuf[offset+1]) & 0x3f) << 6) + (uint32(bytesBuf[offset+2]) & 0x3f)
				if c1 >= 0xD800 && c1 <= 0xDBFF {
					c2, n2, err := decodeUcs2Rune(d.reader)
					if err != nil {
						return s, perrors.WithStack(err)
					}

					c := rune(c1-0xD800)<<10 + (c2 - 0xDC00) + 0x10000
					utf8.EncodeRune(bytesBuf[offset:], c)

					// update next rune
					offset += n2
					chunkLen--
				}

				offset += 3
			} else {
				return s, perrors.Errorf("bad utf-8 encoding, offset=%d\n", offset)
			}

			chunkLen--
		}

		b := bytesBuf[:offset]
		return *(*string)(unsafe.Pointer(&b)), nil
	}

	return s, perrors.Errorf("unknown string tag %#x\n", tag)
}
