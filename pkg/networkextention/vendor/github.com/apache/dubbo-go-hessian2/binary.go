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
	"io"
)

import (
	gxbytes "github.com/dubbogo/gost/bytes"
	perrors "github.com/pkg/errors"
)

/////////////////////////////////////////
// Binary, []byte
/////////////////////////////////////////

// # 8-bit binary data split into 64k chunks
// ::= x41(A) b1 b0 <binary-data> binary # non-final chunk
// ::= x42(B) b1 b0 <binary-data>        # final chunk
// ::= [x20-x2f] <binary-data>           # binary data of length 0-15
// ::= [x34-x37] <binary-data>           # binary data of length 0-1023
func encBinary(b []byte, v []byte) []byte {
	var (
		length  uint16
		vLength int
	)

	if v == nil {
		return encByte(b, BC_NULL)
	}

	vLength = len(v)
	for {
		if vLength > CHUNK_SIZE {
			length = CHUNK_SIZE
			b = encByte(b, BC_BINARY_CHUNK, byte(length>>8), byte(length))
		} else {
			length = uint16(vLength)
			if vLength <= int(BINARY_DIRECT_MAX) {
				b = encByte(b, byte(int(BC_BINARY_DIRECT)+vLength))
			} else if vLength <= int(BINARY_SHORT_MAX) {
				b = encByte(b, byte(int(BC_BINARY_SHORT)+vLength>>8), byte(vLength))
			} else {
				b = encByte(b, BC_BINARY, byte(vLength>>8), byte(vLength))
			}
		}

		b = append(b, v[:length]...)
		v = v[length:]
		vLength = len(v)

		if vLength == 0 {
			break
		}
	}

	return b
}

/////////////////////////////////////////
// Binary, []byte
/////////////////////////////////////////

// # 8-bit binary data split into 64k chunks
// ::= x41('A') b1 b0 <binary-data> binary # non-final chunk
// ::= x42('B') b1 b0 <binary-data>        # final chunk
// ::= [x20-x2f] <binary-data>  # binary data of length 0-15
// ::= [x34-x37] <binary-data>  # binary data of length 0-1023
func (d *Decoder) getBinaryLength(tag byte) (int, error) {
	var (
		err error
		buf [2]byte
	)

	if tag >= BC_BINARY_DIRECT && tag <= INT_DIRECT_MAX { // [0x20, 0x2f]
		return int(tag - BC_BINARY_DIRECT), nil
	}

	if tag >= BC_BINARY_SHORT && tag <= byte(0x37) { // [0x34, 0x37]
		_, err = io.ReadFull(d.reader, buf[:1])
		if err != nil {
			return 0, perrors.WithStack(err)
		}

		return int(tag-BC_BINARY_SHORT)<<8 + int(buf[0]), nil
	}

	if tag != BC_BINARY_CHUNK && tag != BC_BINARY {
		return 0, perrors.Errorf("illegal binary tag:%d", tag)
	}

	_, err = io.ReadFull(d.reader, buf[:2])
	if err != nil {
		return 0, perrors.WithStack(err)
	}

	return int(buf[0])<<8 + int(buf[1]), nil
}

func (d *Decoder) decBinary(flag int32) ([]byte, error) {
	var (
		err    error
		tag    byte
		length int
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, err = d.readBufByte()
		if err != nil {
			return nil, perrors.WithStack(err)
		}
	}

	if tag == BC_NULL {
		return []byte(""), nil
	}

	data := make([]byte, 0, 128)
	bufp := gxbytes.AcquireBytes(65536)
	defer gxbytes.ReleaseBytes(bufp)
	buf := *bufp

	for {
		length, err = d.getBinaryLength(tag)
		if err != nil {
			return nil, perrors.WithStack(err)
		}

		_, err = io.ReadFull(d.reader, buf[:length])
		if err != nil {
			return nil, perrors.WithStack(err)
		}

		data = append(data, buf[:length]...)

		if tag != BC_BINARY_CHUNK {
			break
		}

		tag, err = d.readBufByte()
		if err != nil {
			return nil, perrors.WithStack(err)
		}
	}
	return data, nil
}
