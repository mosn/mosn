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

package gxbig

import (
	"encoding/binary"
	"fmt"
	"math/big"
)

// Integer represents a integer value.
type Integer struct {
	value big.Int

	// used in hessian encoding/decoding
	// You Should not use it in go
	Signum int32
	Mag    []int

	// Deprecated: compatible with java8 serialize
	FirstNonzeroIntNum int
	LowestSetBit       int
	BitLength          int
	BitCount           int
}

func (Integer) JavaClassName() string {
	return "java.math.BigInteger"
}

// FromString set data from a 10-bases number
func (i *Integer) FromString(s string) (err error) {
	_, ok := i.value.SetString(s, 10)
	if !ok {
		err = fmt.Errorf("'%s' is not a 10-based number", s)
	}
	return
}

// FromSignAndMag set data from a array of big-endian unsigned uint32, it's used in hessian decoding
// @see https://docs.oracle.com/javase/8/docs/api/java/math/BigInteger.html#BigInteger-int-byte:A-
func (i *Integer) FromSignAndMag(signum int32, mag []int) {
	if signum == 0 && len(mag) == 0 {
		return
	}

	bytes := make([]byte, 4*len(mag))
	for j := 0; j < len(mag); j++ {
		binary.BigEndian.PutUint32(bytes[j*4:(j+1)*4], uint32(mag[j]))
	}
	i.value.SetBytes(bytes)

	if signum == -1 {
		i.value.Neg(&i.value)
	}
}

// GetSignAndMag is used in hessian encoding
func (i *Integer) GetSignAndMag() (signum int32, mag []int) {
	signum = int32(i.value.Sign())

	bytes := i.value.Bytes()
	outOf4 := len(bytes) % 4
	if outOf4 > 0 {
		bytes = append(make([]byte, 4-outOf4), bytes...)
	}

	size := len(bytes) / 4

	mag = make([]int, size)

	for i := 0; i < size; i++ {
		mag[i] = int(binary.BigEndian.Uint32(bytes[i*4 : (i+1)*4]))
	}

	return
}

func (i *Integer) Value() *big.Int {
	return &i.value
}

func (i *Integer) SetValue(value *big.Int) {
	i.value = *value
}

func (i *Integer) String() string { return i.value.String() }

func (i *Integer) Format(s fmt.State, ch rune) { i.value.Format(s, ch) }

func (i *Integer) GobEncode() ([]byte, error) { return i.value.GobEncode() }
func (i *Integer) GobDecode(buf []byte) error { return i.value.GobDecode(buf) }

func (i *Integer) MarshalText() (text []byte, err error) { return i.value.MarshalText() }
func (i *Integer) UnmarshalText(text []byte) error       { return i.value.UnmarshalText(text) }

func (i *Integer) MarshalJSON() ([]byte, error)    { return i.value.MarshalJSON() }
func (i *Integer) UnmarshalJSON(text []byte) error { return i.value.UnmarshalJSON(text) }
