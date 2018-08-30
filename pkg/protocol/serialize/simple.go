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

package serialize

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"unsafe"
)

// Instance
// singleton of simpleSerialization
var Instance = simpleSerialization{}

type simpleSerialization struct {
}

func (s *simpleSerialization) GetSerialNum() int {
	return 6
}

func (s *simpleSerialization) Serialize(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte{0}, nil
	}

	buf := bytes.Buffer{}

	var err error
	switch v.(type) {
	case string:
		_, err = encodeString(v.(string), &buf)
	case map[string]string:
		err = encodeMap(v.(map[string]string), &buf)
	case []uint8:
		buf.WriteByte(3)
		err = encodeBytes(v.([]uint8), &buf)
	}

	return buf.Bytes(), err
}

func (s *simpleSerialization) DeSerialize(b []byte, v interface{}) (interface{}, error) {
	if len(b) == 0 {
		return nil, nil
	}

	if sv, ok := v.(*string); ok {
		_, err := decodeString(b, sv)

		if err != nil {
			return nil, err
		}

		return sv, err
	}

	if mv, ok := v.(*map[string]string); ok {
		err := decodeMap(b, mv)

		if err != nil {
			return nil, err
		}

		return mv, err
	}

	return nil, nil
}

func (s *simpleSerialization) SerializeMulti(v []interface{}) ([]byte, error) {
	// TODO support multi value
	if len(v) == 0 {
		return nil, nil
	}
	if len(v) == 1 {
		return s.Serialize(v[0])
	}
	return nil, errors.New("do not support multi value in simpleSerialization")
}

func (s *simpleSerialization) DeSerializeMulti(b []byte, v []interface{}) (ret []interface{}, err error) {
	//TODO support multi value
	var rv interface{}
	if v != nil {
		if len(v) == 0 {
			return nil, nil
		}
		if len(v) > 1 {
			return nil, errors.New("do not support multi value in simpleSerialization")
		}
		rv, err = s.DeSerialize(b, v[0])
	} else {
		rv, err = s.DeSerialize(b, nil)
	}
	return []interface{}{rv}, err
}

func readInt32(b []byte) (int, error) {
	if len(b) < 4 {
		return 0, errors.New("no enough bytes")
	}

	return int(binary.BigEndian.Uint32(b[:4])), nil
}

func decodeString(b []byte, result *string) (int, error) {
	*result = string(b)
	return len(b), nil
}

/**
这个map是参考com.alipay.sofa.rpc.remoting.codec.SimpleMapSerializer进行实现的.
*/
func decodeMap(b []byte, result *map[string]string) error {
	totalLen := len(b)
	index := 0

	for index < totalLen {

		length, err := readInt32(b[index:])
		if err != nil {
			return err
		}
		index += 4

		key := b[index : index+length]
		index += length

		length, err = readInt32(b[index:])
		if err != nil {
			return err
		}
		index += 4

		value := b[index : index+length]
		index += length

		(*result)[string(key)] = string(value)
	}
	return nil
}

func encodeString(v string, buf *bytes.Buffer) (int, error) {
	b := unsafeStrToByte(v)

	return buf.Write(b)
}

func encodeMap(v map[string]string, buf *bytes.Buffer) error {
	lenBuf := make([]byte, 4)

	for key, value := range v {

		keyBytes := unsafeStrToByte(key)
		keyLen := len(keyBytes)

		binary.BigEndian.PutUint32(lenBuf, uint32(keyLen))
		buf.Write(lenBuf)
		buf.Write(keyBytes)

		valueBytes := unsafeStrToByte(value)
		valueLen := len(valueBytes)

		binary.BigEndian.PutUint32(lenBuf, uint32(valueLen))
		buf.Write(lenBuf)
		buf.Write(valueBytes)
	}

	return nil
}

func encodeBytes(v []uint8, buf *bytes.Buffer) error {
	l := len(v)
	err := binary.Write(buf, binary.BigEndian, int32(l))
	err = binary.Write(buf, binary.BigEndian, v)
	return err
}

func unsafeStrToByte(s string) []byte {
	var b []byte
	byteHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))

	// we need to take the address of s's Data field and assign it to b's Data field in one
	// expression as it as a uintptr and in the future Go may have a compacting GC that moves
	// pointers but it will not update uintptr values, but single expressions should be safe.
	// For more details see https://groups.google.com/forum/#!msg/golang-dev/rd8XgvAmtAA/p6r28fbF1QwJ
	byteHeader.Data = (*reflect.StringHeader)(unsafe.Pointer(&s)).Data

	// need to take the length of s here to ensure s is live until after we update b's Data
	// field since the garbage collector can collect a variable once it is no longer used
	// not when it goes out of scope, for more details see https://github.com/golang/go/issues/9046
	l := len(s)
	byteHeader.Len = l
	byteHeader.Cap = l

	return b
}
