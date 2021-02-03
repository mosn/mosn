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
	"reflect"
	"time"
)

import (
	perrors "github.com/pkg/errors"
)

/////////////////////////////////////////
// Date
/////////////////////////////////////////

var ZeroDate = time.Time{}

// # time in UTC encoded as 64-bit long milliseconds since epoch
// ::= x4a b7 b6 b5 b4 b3 b2 b1 b0
// ::= x4b b3 b2 b1 b0       # minutes since epoch
func encDateInMs(b []byte, i interface{}) []byte {

	value := UnpackPtrValue(reflect.ValueOf(i))
	vi := value.Interface().(time.Time)
	if vi == ZeroDate {
		b = append(b, BC_NULL)
		return nil
	}
	b = append(b, BC_DATE)
	return append(b, PackInt64(vi.UnixNano()/1e6)...)
}

func encDateInMimute(b []byte, v time.Time) []byte {
	b = append(b, BC_DATE_MINUTE)
	return append(b, PackInt32(int32(v.UnixNano()/60e9))...)
}

/////////////////////////////////////////
// Date
/////////////////////////////////////////

// # time in UTC encoded as 64-bit long milliseconds since epoch
// ::= x4a b7 b6 b5 b4 b3 b2 b1 b0
// ::= x4b b3 b2 b1 b0       # minutes since epoch
func (d *Decoder) decDate(flag int32) (time.Time, error) {
	var (
		err error
		l   int
		tag byte
		buf [8]byte
		s   []byte
		i64 int64
		t   time.Time
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}

	switch {
	case tag == BC_NULL:
		return ZeroDate, nil
	case tag == BC_DATE: //'d': //date
		s = buf[:8]
		l, err = d.next(s)
		if err != nil {
			return t, err
		}
		if l != 8 {
			return t, ErrNotEnoughBuf
		}
		i64 = UnpackInt64(s)
		return time.Unix(i64/1000, i64%1000*10e5), nil
		// return time.Unix(i64/1000, i64*100), nil

	case tag == BC_DATE_MINUTE:
		s = buf[:4]
		l, err = d.next(s)
		if err != nil {
			return t, err
		}
		if l != 4 {
			return t, ErrNotEnoughBuf
		}
		i64 = int64(UnpackInt32(s))
		return time.Unix(i64*60, 0), nil

	default:
		return t, perrors.Errorf("decDate Invalid type: %v", tag)
	}
}
