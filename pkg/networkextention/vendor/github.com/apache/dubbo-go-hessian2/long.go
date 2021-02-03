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
	"encoding/binary"
	"io"
)

import (
	perrors "github.com/pkg/errors"
)

/////////////////////////////////////////
// Int64
/////////////////////////////////////////

// # 64-bit signed long integer
// ::= 'L' b7 b6 b5 b4 b3 b2 b1 b0
// ::= [xd8-xef]             # -x08 to x0f
// ::= [xf0-xff] b0          # -x800 to x7ff
// ::= [x38-x3f] b1 b0       # -x40000 to x3ffff
// ::= x59 b3 b2 b1 b0       # 32-bit integer cast to long
func encInt64(b []byte, v int64) []byte {
	if int64(LONG_DIRECT_MIN) <= v && v <= int64(LONG_DIRECT_MAX) {
		return encByte(b, byte(v+int64(BC_LONG_ZERO)))
	} else if int64(LONG_BYTE_MIN) <= v && v <= int64(LONG_BYTE_MAX) {
		return encByte(b, byte(int64(BC_LONG_BYTE_ZERO)+(v>>8)), byte(v))
	} else if int64(LONG_SHORT_MIN) <= v && v <= int64(LONG_SHORT_MAX) {
		return encByte(b, byte(int64(BC_LONG_SHORT_ZERO)+(v>>16)), byte(v>>8), byte(v))
	} else if -0x80000000 <= v && v <= 0x7fffffff {
		return encByte(b, BC_LONG_INT, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	}

	return encByte(b, 'L', byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

/////////////////////////////////////////
// Int64
/////////////////////////////////////////

// # 64-bit signed long integer
// ::= 'L' b7 b6 b5 b4 b3 b2 b1 b0
// ::= [xd8-xef]             # -x08 to x0f
// ::= [xf0-xff] b0          # -x800 to x7ff
// ::= [x38-x3f] b1 b0       # -x40000 to x3ffff
// ::= x59 b3 b2 b1 b0       # 32-bit integer cast to long
func (d *Decoder) decInt64(flag int32) (int64, error) {
	var (
		err error
		tag byte
		buf [8]byte
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}

	switch {
	case tag == BC_NULL:
		return int64(0), nil

	case tag == BC_FALSE:
		return int64(0), nil

	case tag == BC_TRUE:
		return int64(1), nil

		// direct integer
	case tag >= 0x80 && tag <= 0xbf:
		return int64(tag - BC_INT_ZERO), nil

		// byte int
	case tag >= 0xc0 && tag <= 0xcf:
		if _, err = io.ReadFull(d.reader, buf[:1]); err != nil {
			return 0, perrors.WithStack(err)
		}
		return int64(tag-BC_INT_BYTE_ZERO)<<8 + int64(buf[0]), nil

		// short int
	case tag >= 0xd0 && tag <= 0xd7:
		if _, err = io.ReadFull(d.reader, buf[:2]); err != nil {
			return 0, perrors.WithStack(err)
		}
		return int64(tag-BC_INT_SHORT_ZERO)<<16 + int64(buf[0])<<8 + int64(buf[1]), nil

	case tag == BC_DOUBLE_BYTE:
		tag, _ = d.readByte()
		return int64(tag), nil

	case tag == BC_DOUBLE_SHORT:
		if _, err = io.ReadFull(d.reader, buf[:2]); err != nil {
			return 0, perrors.WithStack(err)
		}

		return int64(int(buf[0])<<8 + int(buf[1])), nil

	case tag == BC_INT: // 'I'
		i32, err := d.decInt32(TAG_READ)
		return int64(i32), err

	case tag == BC_LONG_INT:
		var i32 int32
		err = binary.Read(d.reader, binary.BigEndian, &i32)
		return int64(i32), perrors.WithStack(err)

	case tag >= 0xd8 && tag <= 0xef:
		i8 := int8(tag - BC_LONG_ZERO)
		return int64(i8), nil

	case tag >= 0xf0 && tag <= 0xff:
		buf := []byte{tag - BC_LONG_BYTE_ZERO, 0}
		_, err = io.ReadFull(d.reader, buf[1:])
		if err != nil {
			return 0, perrors.WithStack(err)
		}
		u16 := binary.BigEndian.Uint16(buf)
		i16 := int16(u16)
		return int64(i16), nil

	case tag >= 0x38 && tag <= 0x3f:
		// Use int32 to represent int24.
		buf := []byte{0, tag - BC_LONG_SHORT_ZERO, 0, 0}
		if buf[1]&0x80 != 0 {
			buf[0] = 0xff
		}
		_, err = io.ReadFull(d.reader, buf[2:])
		if err != nil {
			return 0, perrors.WithStack(err)
		}
		u32 := binary.BigEndian.Uint32(buf)
		i32 := int32(u32)
		return int64(i32), nil

	case tag == BC_LONG:
		var i64 int64
		err = binary.Read(d.reader, binary.BigEndian, &i64)
		return i64, perrors.WithStack(err)

	case tag == BC_DOUBLE_ZERO:
		return int64(0), nil

	case tag == BC_DOUBLE_ONE:
		return int64(1), nil

	case tag == BC_DOUBLE_MILL:
		i64, err := d.decInt32(TAG_READ)
		return int64(i64), perrors.WithStack(err)

	default:
		return 0, perrors.Errorf("decInt64 long wrong tag:%#x", tag)
	}
}
