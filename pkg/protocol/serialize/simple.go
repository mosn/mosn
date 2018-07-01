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

//singleton
var Instance = SimpleSerialization{}

type SimpleSerialization struct {
}

func (s *SimpleSerialization) GetSerialNum() int {
	return 6
}

func (s *SimpleSerialization) Serialize(v interface{}) ([]byte, error) {
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

func (s *SimpleSerialization) DeSerialize(b []byte, v interface{}) (interface{}, error) {
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

func (s *SimpleSerialization) SerializeMulti(v []interface{}) ([]byte, error) {
	// TODO support multi value
	if len(v) == 0 {
		return nil, nil
	}
	if len(v) == 1 {
		return s.Serialize(v[0])
	}
	return nil, errors.New("do not support multi value in SimpleSerialization")
}

func (s *SimpleSerialization) DeSerializeMulti(b []byte, v []interface{}) (ret []interface{}, err error) {
	//TODO support multi value
	var rv interface{}
	if v != nil {
		if len(v) == 0 {
			return nil, nil
		}
		if len(v) > 1 {
			return nil, errors.New("do not support multi value in SimpleSerialization")
		}
		rv, err = s.DeSerialize(b, v[0])
	} else {
		rv, err = s.DeSerialize(b, nil)
	}
	return []interface{}{rv}, err
}

func readInt32(buf *bytes.Buffer) (int, error) {
	var i int32
	err := binary.Read(buf, binary.BigEndian, &i)
	if err != nil {
		return 0, err
	}
	return int(i), nil
}

func decodeString(b []byte, result *string) (int, error) {
	*result = string(b)

	return len(b), nil
}

/**
这个map是参考com.alipay.sofa.rpc.remoting.codec.SimpleMapSerializer进行实现的.
*/
func decodeMap(b []byte, result *map[string]string) error {
	buf := bytes.NewBuffer(b)

	for {
		length, err := readInt32(buf)
		if length == -1 || err != nil {
			return nil
		}

		key := make([]byte, length)
		buf.Read(key)

		length, err = readInt32(buf)
		if length == -1 || err != nil {
			return nil
		}

		value := make([]byte, length)
		buf.Read(value)

		keyStr := string(key)
		valueStr := string(value)

		(*result)[keyStr] = valueStr
	}
}

func encodeStringMap(v string, buf *bytes.Buffer) (int32, error) {
	b := unsafeStrToByte(v)
	l := int32(len(b))
	err := binary.Write(buf, binary.BigEndian, l)
	err = binary.Write(buf, binary.BigEndian, b)
	if err != nil {
		return 0, err
	}
	return l + 4, nil
}

func encodeString(v string, buf *bytes.Buffer) (int32, error) {
	b := unsafeStrToByte(v)
	l := int32(len(b))

	err := binary.Write(buf, binary.BigEndian, b)

	if err != nil {
		return 0, err
	}

	return l + 4, nil
}

func encodeMap(v map[string]string, buf *bytes.Buffer) error {
	b := bytes.Buffer{}

	var size, l int32
	var err error
	for k, v := range v {
		l, err = encodeStringMap(k, &b)
		size += l
		if err != nil {
			return err
		}
		l, err = encodeStringMap(v, &b)
		size += l
		if err != nil {
			return err
		}
	}
	err = binary.Write(buf, binary.BigEndian, b.Bytes()[:size])

	return err
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
