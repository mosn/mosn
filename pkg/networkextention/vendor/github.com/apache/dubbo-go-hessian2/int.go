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
	"reflect"
)

import (
	perrors "github.com/pkg/errors"
)

/////////////////////////////////////////
// Int32
/////////////////////////////////////////

// # 32-bit signed integer
// ::= 'I' b3 b2 b1 b0
// ::= [x80-xbf]             # -x10 to x3f
// ::= [xc0-xcf] b0          # -x800 to x7ff
// ::= [xd0-xd7] b1 b0       # -x40000 to x3ffff
func encInt32(b []byte, v int32) []byte {
	if int32(INT_DIRECT_MIN) <= v && v <= int32(INT_DIRECT_MAX) {
		return encByte(b, byte(v+int32(BC_INT_ZERO)))
	} else if int32(INT_BYTE_MIN) <= v && v <= int32(INT_BYTE_MAX) {
		return encByte(b, byte(int32(BC_INT_BYTE_ZERO)+v>>8), byte(v))
	} else if int32(INT_SHORT_MIN) <= v && v <= int32(INT_SHORT_MAX) {
		return encByte(b, byte(v>>16+int32(BC_INT_SHORT_ZERO)), byte(v>>8), byte(v))
	}

	return encByte(b, byte('I'), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

/////////////////////////////////////////
// Int32
/////////////////////////////////////////

// # 32-bit signed integer
// ::= 'I' b3 b2 b1 b0
// ::= [x80-xbf]             # -x10 to x3f
// ::= [xc0-xcf] b0          # -x800 to x7ff
// ::= [xd0-xd7] b1 b0       # -x40000 to x3ffff
func (d *Decoder) decInt32(flag int32) (int32, error) {
	var (
		err error
		tag byte
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}

	switch {
	case tag >= 0x80 && tag <= 0xbf:
		i8 := int8(tag - BC_INT_ZERO)
		return int32(i8), nil

	case tag >= 0xc0 && tag <= 0xcf:
		buf := []byte{tag - BC_INT_BYTE_ZERO, 0}
		_, err = io.ReadFull(d.reader, buf[1:])
		if err != nil {
			return 0, perrors.WithStack(err)
		}
		u16 := binary.BigEndian.Uint16(buf)
		i16 := int16(u16)
		return int32(i16), nil

	case tag >= 0xd0 && tag <= 0xd7:
		// Use int32 to represent int24.
		buf := []byte{0, tag - BC_INT_SHORT_ZERO, 0, 0}
		if buf[1]&0x80 != 0 {
			buf[0] = 0xff
		}
		_, err = io.ReadFull(d.reader, buf[2:])
		if err != nil {
			return 0, perrors.WithStack(err)
		}
		u32 := binary.BigEndian.Uint32(buf)
		return int32(u32), nil

	case tag == BC_INT:
		var i32 int32
		err = binary.Read(d.reader, binary.BigEndian, &i32)
		return i32, perrors.WithStack(err)

	default:
		return 0, perrors.Errorf("decInt32 integer wrong tag:%#x", tag)
	}
}

func (d *Encoder) encTypeInt32(b []byte, p interface{}) ([]byte, error) {
	value := reflect.ValueOf(p)
	if PackPtr(value).IsNil() {
		return encNull(b), nil
	}
	value = UnpackPtrValue(value)
	if value.Kind() != reflect.Int32 {
		return nil, perrors.Errorf("encode reflect Int32 integer wrong, it's not int32 pointer")
	}
	return encInt32(b, int32(value.Int())), nil
}
