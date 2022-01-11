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
	"strings"
)

func init() {
	SetCollectionSerialize(&IntegerArray{})
	SetCollectionSerialize(&ByteArray{})
	SetCollectionSerialize(&ShortArray{})
	SetCollectionSerialize(&BooleanArray{})
	SetCollectionSerialize(&LongArray{})
	SetCollectionSerialize(&FloatArray{})
	SetCollectionSerialize(&DoubleArray{})
	SetCollectionSerialize(&CharacterArray{})
}

// BooleanArray Boolean[]
type BooleanArray struct {
	Values []bool
}

// nolint
func (ba *BooleanArray) Get() []interface{} {
	res := make([]interface{}, len(ba.Values))
	for i, v := range ba.Values {
		res[i] = v
	}
	return res
}

// nolint
func (ba *BooleanArray) Set(vs []interface{}) {
	values := make([]bool, len(vs))
	for i, v := range vs {
		values[i] = v.(bool)
	}
	ba.Values = values
}

// nolint
func (*BooleanArray) JavaClassName() string {
	return "[java.lang.Boolean"
}

//IntegerArray Integer[]
type IntegerArray struct {
	Values []int32
}

// nolint
func (ia *IntegerArray) Get() []interface{} {
	res := make([]interface{}, len(ia.Values))
	for i, v := range ia.Values {
		res[i] = v
	}
	return res
}

// nolint
func (ia *IntegerArray) Set(vs []interface{}) {
	values := make([]int32, len(vs))
	for i, v := range vs {
		values[i] = v.(int32)
	}
	ia.Values = values
}

// nolint
func (*IntegerArray) JavaClassName() string {
	return "[java.lang.Integer"
}

// ByteArray Byte[]
type ByteArray struct {
	Values []uint8
}

// nolint
func (ba *ByteArray) Get() []interface{} {
	res := make([]interface{}, len(ba.Values))
	for i, v := range ba.Values {
		res[i] = v
	}
	return res
}

// nolint
func (ba *ByteArray) Set(vs []interface{}) {
	values := make([]uint8, len(vs))
	for i, v := range vs {
		values[i] = uint8(v.(int32))
	}
	ba.Values = values
}

// nolint
func (*ByteArray) JavaClassName() string {
	return "[java.lang.Byte"
}

// ShortArray Short[]
type ShortArray struct {
	Values []int16
}

// nolint
func (sa *ShortArray) Get() []interface{} {
	res := make([]interface{}, len(sa.Values))
	for i, v := range sa.Values {
		res[i] = v
	}
	return res
}

// nolint
func (sa *ShortArray) Set(vs []interface{}) {
	values := make([]int16, len(vs))
	for i, v := range vs {
		values[i] = int16(v.(int32))
	}
	sa.Values = values
}

// nolint
func (*ShortArray) JavaClassName() string {
	return "[java.lang.Short"
}

// LongArray Long[]
type LongArray struct {
	Values []int64
}

// nolint
func (ba *LongArray) Get() []interface{} {
	res := make([]interface{}, len(ba.Values))
	for i, v := range ba.Values {
		res[i] = v
	}
	return res
}

// nolint
func (ba *LongArray) Set(vs []interface{}) {
	values := make([]int64, len(vs))
	for i, v := range vs {
		values[i] = v.(int64)
	}
	ba.Values = values
}

// nolint
func (*LongArray) JavaClassName() string {
	return "[java.lang.Long"
}

// FloatArray Float[]
type FloatArray struct {
	Values []float32
}

// nolint
func (fa *FloatArray) Get() []interface{} {
	res := make([]interface{}, len(fa.Values))
	for i, v := range fa.Values {
		res[i] = v
	}
	return res
}

// nolint
func (fa *FloatArray) Set(vs []interface{}) {
	values := make([]float32, len(vs))
	for i, v := range vs {
		values[i] = float32(v.(float64))
	}
	fa.Values = values
}

// nolint
func (*FloatArray) JavaClassName() string {
	return "[java.lang.Float"
}

// DoubleArray Double[]
type DoubleArray struct {
	Values []float64
}

// nolint
func (da *DoubleArray) Get() []interface{} {
	res := make([]interface{}, len(da.Values))
	for i, v := range da.Values {
		res[i] = v
	}
	return res
}

// nolint
func (da *DoubleArray) Set(vs []interface{}) {
	values := make([]float64, len(vs))
	for i, v := range vs {
		values[i] = v.(float64)
	}
	da.Values = values
}

// nolint
func (*DoubleArray) JavaClassName() string {
	return "[java.lang.Double"
}

// CharacterArray Character[]
type CharacterArray struct {
	Values string
}

// nolint
func (ca *CharacterArray) Get() []interface{} {
	length := len(ca.Values)
	charArr := strings.Split(ca.Values, "")
	res := make([]interface{}, length)
	for i := 0; i < length; i++ {
		res[i] = charArr[i]
	}
	return res
}

// nolint
func (ca *CharacterArray) Set(vs []interface{}) {
	for _, v := range vs {
		ca.Values = ca.Values + v.(string)
	}
}

// nolint
func (*CharacterArray) JavaClassName() string {
	return "[java.lang.Character"
}
