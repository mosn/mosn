// Copyright (c) 2016 ~ 2018, Alex Stocks.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hessian

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"
)

// go test -v encode_decode_test.go encode.go decode.go const.go codec.go pojo.go
var assert = func(want, got []byte, t *testing.T) {
	if !bytes.Equal(want, got) {
		t.Fatalf("want %v , got %v", want, got)
	}
}

func TestEncNull(t *testing.T) {
	var (
		e *Encoder
	)

	e = NewEncoder()
	e.Encode(nil)
	if e.Buffer() == nil {
		t.Fail()
	}
	t.Logf("nil enc result:%s\n", string(e.buffer))
}

func TestEncBool(t *testing.T) {
	var (
		e    *Encoder
		want []byte
	)

	e = NewEncoder()
	e.Encode(true)
	if e.Buffer()[0] != 'T' {
		t.Fail()
	}
	want = []byte{0x54}
	assert(want, e.Buffer(), t)

	e = NewEncoder()
	e.Encode(false)
	if e.Buffer()[0] != 'F' {
		t.Fail()
	}
	want = []byte{0x46}
	assert(want, e.Buffer(), t)
}

func TestEncInt32Len1B(t *testing.T) {
	var (
		v   int32
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	v = 0xe6
	// var v int32 = 0xf016
	e = NewEncoder()
	e.Encode(v)
	if len(e.Buffer()) == 0 {
		t.Fail()
	}
	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	t.Logf("decode(%v) = %v, %v\n", v, res, err)
}

func TestEncInt32Len2B(t *testing.T) {
	var (
		v   int32
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	v = 0xf016
	e = NewEncoder()
	e.Encode(v)
	if len(e.buffer) == 0 {
		t.Fail()
	}
	t.Logf("%#v\n", e.buffer)
	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	t.Logf("decode(%#x) = %#x, %v\n", v, res, err)
}

func TestEncInt32Len4B(t *testing.T) {
	var (
		v   int32
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	v = 0x20161024
	e.Encode(v)
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	t.Logf("decode(%v) = %v, %v\n", v, res, err)
}

func TestEncInt64Len1BDirect(t *testing.T) {
	var (
		v   int64
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	v = 0x1
	e.Encode(int64(v))
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	t.Logf("decode(int64(%#x)) = %#x, %v\n", v, res, err)
}

func TestEncInt64Len1B(t *testing.T) {
	var (
		v   int64
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	v = 0xf6
	e.Encode(int64(v))
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	t.Logf("decode(int64(%#x)) = %#x, %v\n", v, res, err)
}

func TestEncInt64Len2B(t *testing.T) {
	var (
		v   int64
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	v = 0x2016
	e.Encode(int64(v))
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	t.Logf("decode(int64(%#x)) = %#x, %v\n", v, res, err)
}

func TestEncInt64Len3B(t *testing.T) {
	var (
		v   int64
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	v = 101910 // 0x18e16
	e.Encode(int64(v))
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	t.Logf("decode(int64(%#x)) = %#x, %v\n", v, res, err)
}

func TestEncInt64Len8B(t *testing.T) {
	var (
		v   int64
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	v = 0x20161024114530
	e.Encode(int64(v))
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	t.Logf("decode(int64(%#x)) = %#x, %v\n", v, res, err)
}

func TestEncDate(t *testing.T) {
	var (
		v   string
		tz  time.Time
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	v = "2014-02-09 06:15:23"
	tz, _ = time.Parse("2006-01-02 15:04:05", v)
	e.Encode(tz)
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	t.Logf("decode(%s, %s) = %v, %v\n", v, tz.Local(), res, err)
}

func TestEncDouble(t *testing.T) {
	var (
		v   float64
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	v = 2016.1024
	e.Encode(v)
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	t.Logf("decode(%v) = %v, %v\n", v, res, err)
}

func TestEncString(t *testing.T) {
	var (
		v   string
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	v = "hello"
	e.Encode(v)
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	t.Logf("decode(%v) = %v, %v\n", v, res, err)
}

func TestEncShortRune(t *testing.T) {
	var (
		v   string
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	v = "我化尘埃飞扬，追寻赤裸逆翔"
	e.Encode(v)
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	t.Logf("decode(%v) = %v, %v\n", v, res, err)
}

func TestEncRune(t *testing.T) {
	var (
		v   string
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	v = "我化尘埃飞扬，追寻赤裸逆翔, 奔去七月刑场，时间烧灼滚烫, 回忆撕毁臆想，路上行走匆忙, 难能可贵世上，散播留香磁场, 我欲乘风破浪，踏遍黄沙海洋, 与其误会一场，也要不负勇往, 我愿你是个谎，从未出现南墙, 笑是神的伪装，笑是强忍的伤, 我想你就站在，站在大漠边疆, 我化尘埃飞扬，追寻赤裸逆翔," +
		" 奔去七月刑场，时间烧灼滚烫, 回忆撕毁臆想，路上行走匆忙, 难能可贵世上，散播留香磁场, 我欲乘风破浪，踏遍黄沙海洋, 与其误会一场，也要不负勇往, 我愿你是个谎，从未出现南墙, 笑是神的伪装，笑是强忍的伤, 我想你就站在，站在大漠边疆."
	v = v + v + v + v + v
	v = v + v + v + v + v
	v = v + v + v + v + v
	v = v + v + v + v + v
	v = v + v + v + v + v
	fmt.Printf("vlen:%d\n", len(v))
	e.Encode(v)
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	if err != nil {
		t.Errorf("Decode() = %v", err)
	}
	// t.Logf("decode(%v) = %v, %v\n", v, res, err)
	assert([]byte(res.(string)), []byte(v), t)
}

func TestEncBinary(t *testing.T) {
	var (
		v   []byte
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	v = []byte{}
	e.Encode(v)
	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	if err != nil {
		t.Errorf("Decode() = %v", err)
	}
	t.Logf("decode(%v) = %v, %v\n", v, res, err)

	v = []byte{10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 'a', 'b', 'c', 'd'}
	e = NewEncoder()
	e.Encode(v)
	t.Logf("encode(%v) = %v\n", v, e.Buffer())
	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	if err != nil {
		t.Errorf("Decode() = %v", err)
	}
	t.Logf("decode(%v) = %v, %v, equal:%v\n", v, res, err, bytes.Equal(v, res.([]byte)))
	assert(v, res.([]byte), t)
}

func TestEncBinaryShort(t *testing.T) {
	var (
		v   [1010]byte
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	for i := 0; i < len(v); i++ {
		v[i] = byte(i % 123)
	}

	e = NewEncoder()
	e.Encode(v[:])
	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	if err != nil {
		t.Errorf("Decode() = %v", err)
	}
	assert(v[:], res.([]byte), t)
}

func TestEncBinaryChunk(t *testing.T) {
	var (
		v   [65530]byte
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	for i := 0; i < len(v); i++ {
		v[i] = byte(i % 123)
	}

	e = NewEncoder()
	e.Encode(v[:])
	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	if err != nil {
		t.Errorf("Decode() = %v", err)
	}
	assert(v[:], res.([]byte), t)
}

func TestEncList(t *testing.T) {
	var (
		list []interface{}
		err  error
		e    *Encoder
		d    *Decoder
		res  interface{}
	)

	e = NewEncoder()
	list = []interface{}{100, 10.001, "hello", []byte{0, 2, 4, 6, 8, 10}, true, nil, false}
	e.Encode(list)
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	if err != nil {
		t.Errorf("Decode() = %v", err)
	}
	t.Logf("decode(%v) = %v, %v\n", list, res, err)
}

func TestEncUntypedMap(t *testing.T) {
	var (
		m   map[interface{}]interface{}
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	m = make(map[interface{}]interface{})
	m["hello"] = "world"
	m[100] = "100"
	m[100.1010] = 101910
	m[true] = true
	m[false] = true
	e.Encode(m)
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	if err != nil {
		t.Errorf("Decode() = %v", err)
	}
	t.Logf("decode(%v) = %v, %v\n", m, res, err)
}

func TestEncTypedMap(t *testing.T) {
	var (
		m   map[int]string
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	m = make(map[int]string)
	m[0] = "hello"
	m[1] = "golang"
	m[2] = "world"
	e.Encode(m)
	if len(e.Buffer()) == 0 {
		t.Fail()
	}

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	if err != nil {
		t.Errorf("Decode() = %v", err)
	}
	t.Logf("decode(%v) = %v, %v\n", m, res, err)
}

type Department struct {
	Name string
}

func (Department) JavaClassName() string {
	return "com.bdt.info.Department"
}

type WorkerInfo struct {
	Name           string
	Addrress       string
	Age            int
	Salary         float32
	Payload        map[string]int32
	FalimyMemebers []string
	Dpt            Department
}

func (WorkerInfo) JavaClassName() string {
	return "com.bdt.info.WorkerInfo"
}

func TestEncEmptyStruct(t *testing.T) {
	var (
		w   WorkerInfo
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	w = WorkerInfo{
		Name:           "Trump",
		Addrress:       "W,D.C.",
		Age:            72,
		Salary:         21000.03,
		Payload:        map[string]int32{"Number": 2017061118},
		FalimyMemebers: []string{"m1", "m2", "m3"},
		// Dpt: Department{
		// 	Name: "Adm",
		// },
	}
	e.Encode(w)

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	if err != nil {
		t.Errorf("Decode() = %v", err)
	}
	t.Logf("decode(%v) = %v, %v\n", w, res, err)

	reflect.DeepEqual(w, res)
}

func TestEncStruct(t *testing.T) {
	var (
		w   WorkerInfo
		err error
		e   *Encoder
		d   *Decoder
		res interface{}
	)

	e = NewEncoder()
	w = WorkerInfo{
		Name:           "Trump",
		Addrress:       "W,D.C.",
		Age:            72,
		Salary:         21000.03,
		Payload:        map[string]int32{"Number": 2017061118},
		FalimyMemebers: []string{"m1", "m2", "m3"},
		Dpt: Department{
			Name: "Adm",
		},
	}
	e.Encode(w)

	d = NewDecoder(e.Buffer())
	res, err = d.Decode()
	if err != nil {
		t.Errorf("Decode() = %v", err)
	}
	t.Logf("decode(%v) = %v, %v\n", w, res, err)

	reflect.DeepEqual(w, res)
}
