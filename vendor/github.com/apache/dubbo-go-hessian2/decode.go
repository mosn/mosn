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
	"io"
	"reflect"
)

import (
	perrors "github.com/pkg/errors"
)

// Decoder struct
type Decoder struct {
	reader *bufio.Reader
	refs   []interface{}
	// record type refs, both list and map need it
	typeRefs      *TypeRefs
	classInfoList []*classInfo
	isSkip        bool
}

// Error part
var (
	ErrNotEnoughBuf    = perrors.Errorf("not enough buf")
	ErrIllegalRefIndex = perrors.Errorf("illegal ref index")
)

// NewDecoder generate a decoder instance
func NewDecoder(b []byte) *Decoder {
	return &Decoder{reader: bufio.NewReader(bytes.NewReader(b)), typeRefs: &TypeRefs{records: map[string]bool{}}}
}

// NewDecoderSize generate a decoder instance.
func NewDecoderSize(b []byte, size int) *Decoder {
	return &Decoder{reader: bufio.NewReaderSize(bytes.NewReader(b), size), typeRefs: &TypeRefs{records: map[string]bool{}}}
}

// NewDecoderWithSkip generate a decoder instance with skip.
func NewDecoderWithSkip(b []byte) *Decoder {
	return &Decoder{reader: bufio.NewReader(bytes.NewReader(b)), typeRefs: &TypeRefs{records: map[string]bool{}}, isSkip: true}
}

// NewCheapDecoderWithSkip generate a decoder instance with skip,
// only for cache pool, before decode Reset should be called.
// For example, with pooling use, will effectively improve performance
//
//	var hessianPool = &sync.Pool{
//		New: func() interface{} {
//			return hessian.NewCheapDecoderWithSkip([]byte{})
//		},
//	}
//
//	decoder := hessianPool.Get().(*hessian.Decoder)
//	fill decode data
//	decoder.Reset(data[:])
//  decode anything ...
//	hessianPool.Put(decoder)
func NewCheapDecoderWithSkip(b []byte) *Decoder {
	return &Decoder{reader: bufio.NewReader(bytes.NewReader(b)), isSkip: true}
}

// Clean clean the Decoder (room) for a new object decoding.
// Notice it won't reset reader buffer and will continue to read data from it.
func (d *Decoder) Clean() {
	d.typeRefs = &TypeRefs{records: map[string]bool{}}
	d.refs = nil
	d.classInfoList = nil
}

/////////////////////////////////////////
// utilities
/////////////////////////////////////////

func (d *Decoder) Reset(b []byte) *Decoder {
	// reuse reader buf, avoid allocate
	d.reader.Reset(bytes.NewReader(b))
	d.Clean()
	return d
}

// peek a byte
func (d *Decoder) peekByte() byte {
	return d.peek(1)[0]
}

// get the buffer length
func (d *Decoder) len() int {
	d.peek(1) // peek one byte to get the buffer length
	return d.reader.Buffered()
}

// ReadByte read a byte from Decoder, advance the ptr
func (d *Decoder) ReadByte() (byte, error) {
	return d.reader.ReadByte()
}

// Discard skips the next n bytes
func (d *Decoder) Discard(n int) (int, error) {
	return d.reader.Discard(n)
}

// unread a byte
func (d *Decoder) unreadByte() error {
	return d.reader.UnreadByte()
}

// read byte arr, and return the length of b
func (d *Decoder) next(b []byte) (int, error) {
	return d.reader.Read(b)
}

// read byte arr, and return the real length of b
func (d *Decoder) nextFull(b []byte) (int, error) {
	return io.ReadFull(d.reader, b)
}

// peek n bytes, will not advance the read ptr
func (d *Decoder) peek(n int) []byte {
	b, _ := d.reader.Peek(n)
	return b
}

// read utf8 len(s) of array
func (d *Decoder) nextRune(s []rune) []rune {
	var (
		n   int
		i   int
		r   rune
		ri  int
		err error
	)

	n = len(s)
	s = s[:0]
	for i = 0; i < n; i++ {
		if r, ri, err = d.reader.ReadRune(); err == nil && ri > 0 {
			s = append(s, r)
		}
	}

	return s
}

// read the type of data, used to decode list or map
func (d *Decoder) decMapType() (reflect.Type, error) {
	var (
		err     error
		arr     [1]byte
		buf     []byte
		tag     byte
		idx     int32
		typ     reflect.Type
		typName string
	)

	buf = arr[:1]
	if _, err = io.ReadFull(d.reader, buf); err != nil {
		return nil, perrors.WithStack(err)
	}
	tag = buf[0]
	if (tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX) ||
		(tag >= 0x30 && tag <= 0x33) || (tag == BC_STRING) || (tag == BC_STRING_CHUNK) {
		typName, err = d.decString(int32(tag))
		if err != nil {
			return nil, perrors.WithStack(err)
		}

		info, ok := getStructInfo(typName)
		if ok {
			typ = info.typ
		} else {
			typ = reflect.TypeOf(map[interface{}]interface{}{})
		}

		// add to type map
		d.typeRefs.appendTypeRefs(typName, typ)

		return typ, nil
	}

	if idx, err = d.decInt32(int32(tag)); err != nil {
		return nil, perrors.WithStack(err)
	}

	typ = d.typeRefs.Get(int(idx))
	if typ == nil {
		return nil, perrors.Errorf("the type ref index %d is out of range", idx)
	}

	return typ, err
}

// Decode parse hessian data, and ensure the reflection value unpacked
func (d *Decoder) Decode() (interface{}, error) {
	return EnsureInterface(d.DecodeValue())
}

func (d *Decoder) Buffered() int { return d.reader.Buffered() }

// DecodeValue parse hessian data, the return value maybe a reflection value when it's a map, list, object, or ref.
func (d *Decoder) DecodeValue() (interface{}, error) {
	var (
		err error
		tag byte
	)

	tag, err = d.ReadByte()
	if perrors.Is(err, io.EOF) {
		return nil, err
	}

	switch {
	case tag == BC_END:
		// return EOF error for end flag 'Z'
		return nil, io.EOF

	case tag == BC_NULL: // 'N': //null
		return nil, nil

	case tag == BC_TRUE: // 'T': //true
		return true, nil

	case tag == BC_FALSE: //'F': //false
		return false, nil

	case tag == BC_REF: // 'R': //ref, a int which represents the previous list or map
		return d.decRef(int32(tag))

	case (0x80 <= tag && tag <= 0xbf) || (0xc0 <= tag && tag <= 0xcf) ||
		(0xd0 <= tag && tag <= 0xd7) || tag == BC_INT: //'I': //int
		return d.decInt32(int32(tag))

	case (tag >= 0xd8 && tag <= 0xef) || (tag >= 0xf0 && tag <= 0xff) ||
		(tag >= 0x38 && tag <= 0x3f) || (tag == BC_LONG_INT) || (tag == BC_LONG): //'L': //long
		return d.decInt64(int32(tag))

	case (tag == BC_DATE_MINUTE) || (tag == BC_DATE): //'d': //date
		return d.decDate(int32(tag))

	case (tag == BC_DOUBLE_ZERO) || (tag == BC_DOUBLE_ONE) || (tag == BC_DOUBLE_BYTE) ||
		(tag == BC_DOUBLE_SHORT) || (tag == BC_DOUBLE_MILL) || (tag == BC_DOUBLE): //'D': //double
		return d.decDouble(int32(tag))

	// case 'S', 's', 'X', 'x': //string,xml
	case (tag == BC_STRING_CHUNK || tag == BC_STRING) ||
		(tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX) ||
		(tag >= 0x30 && tag <= 0x33):
		return d.decString(int32(tag))

		// case 'B', 'b': //binary
	case (tag == BC_BINARY) || (tag == BC_BINARY_CHUNK) || (tag >= 0x20 && tag <= 0x2f) ||
		(tag >= BC_BINARY_SHORT && tag <= 0x3f):
		return d.decBinary(int32(tag))

	// case 'V': //list
	case (tag >= BC_LIST_DIRECT && tag <= 0x77) || (tag == BC_LIST_FIXED || tag == BC_LIST_VARIABLE) ||
		(tag >= BC_LIST_DIRECT_UNTYPED && tag <= 0x7f) ||
		(tag == BC_LIST_FIXED_UNTYPED || tag == BC_LIST_VARIABLE_UNTYPED):
		return d.decList(int32(tag))

	case (tag == BC_MAP) || (tag == BC_MAP_UNTYPED):
		return d.decMap(int32(tag))

	case (tag == BC_OBJECT_DEF) || (tag == BC_OBJECT) ||
		(BC_OBJECT_DIRECT <= tag && tag <= (BC_OBJECT_DIRECT+OBJECT_DIRECT_MAX)):
		return d.decObject(int32(tag))

	default:
		return nil, perrors.Errorf("Invalid type: %v,>>%v<<<", string(tag), d.peek(d.len()))
	}
}

/////////////////////////////////////////
// typeRefs
/////////////////////////////////////////
type TypeRefs struct {
	typeRefs []reflect.Type
	records  map[string]bool // record if existing for type
}

// appendTypeRefs add list or map type ref
func (t *TypeRefs) appendTypeRefs(name string, p reflect.Type) {
	if t.records[name] {
		return
	}
	t.records[name] = true
	t.typeRefs = append(t.typeRefs, p)
}

func (t *TypeRefs) Get(index int) reflect.Type {
	if len(t.typeRefs) <= index {
		return nil
	}
	return t.typeRefs[index]
}
