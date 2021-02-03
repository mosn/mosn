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
	"math"
)

import (
	perrors "github.com/pkg/errors"
)

/////////////////////////////////////////
// Double
/////////////////////////////////////////

// # 64-bit IEEE double
// ::= 'D' b7 b6 b5 b4 b3 b2 b1 b0
// ::= x5b                   # 0.0
// ::= x5c                   # 1.0
// ::= x5d b0                # byte cast to double (-128.0 to 127.0)
// ::= x5e b1 b0             # short cast to double
// ::= x5f b3 b2 b1 b0       # 32-bit float cast to double
func encFloat(b []byte, v float64) []byte {
	fv := float64(int64(v))
	if fv == v {
		iv := int64(v)
		switch iv {
		case 0:
			return encByte(b, BC_DOUBLE_ZERO)
		case 1:
			return encByte(b, BC_DOUBLE_ONE)
		}
		if iv >= -0x80 && iv < 0x80 {
			return encByte(b, BC_DOUBLE_BYTE, byte(iv))
		} else if iv >= -0x8000 && iv < 0x8000 {
			return encByte(b, BC_DOUBLE_SHORT, byte(iv>>8), byte(iv))
		}

		goto END
	}

END:
	bits := math.Float64bits(v)
	return encByte(b, BC_DOUBLE, byte(bits>>56), byte(bits>>48), byte(bits>>40),
		byte(bits>>32), byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits))
}

/////////////////////////////////////////
// Double
/////////////////////////////////////////

// # 64-bit IEEE double
// ::= 'D' b7 b6 b5 b4 b3 b2 b1 b0
// ::= x5b                   # 0.0
// ::= x5c                   # 1.0
// ::= x5d b0                # byte cast to double (-128.0 to 127.0)
// ::= x5e b1 b0             # short cast to double
// ::= x5f b3 b2 b1 b0       # 32-bit float cast to double
func (d *Decoder) decDouble(flag int32) (interface{}, error) {
	var (
		err error
		tag byte
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}
	switch tag {
	case BC_LONG_INT:
		return d.decInt32(TAG_READ)

	case BC_DOUBLE_ZERO:
		return float64(0), nil

	case BC_DOUBLE_ONE:
		return float64(1), nil

	case BC_DOUBLE_BYTE:
		var i8 int8
		err = binary.Read(d.reader, binary.BigEndian, &i8)
		return float64(i8), perrors.WithStack(err)

	case BC_DOUBLE_SHORT:
		var i16 int16
		err = binary.Read(d.reader, binary.BigEndian, &i16)
		return float64(i16), perrors.WithStack(err)

	case BC_DOUBLE_MILL:
		var i32 int32
		err = binary.Read(d.reader, binary.BigEndian, &i32)
		return float64(i32) / 1000, perrors.WithStack(err)

	case BC_DOUBLE:
		var f64 float64
		err = binary.Read(d.reader, binary.BigEndian, &f64)
		return f64, perrors.WithStack(err)
	}

	return nil, perrors.Errorf("decDouble parse double wrong tag:%d-%#x", int(tag), tag)
}
