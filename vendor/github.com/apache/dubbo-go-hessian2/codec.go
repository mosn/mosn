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

// pack/unpack fixed length variable
package hessian

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"strings"
)

import (
	perrors "github.com/pkg/errors"
)

var (
	_zeroBoolPinter *bool
	_zeroValue      = reflect.ValueOf(_zeroBoolPinter).Elem()
)

func encByte(b []byte, t ...byte) []byte {
	return append(b, t...)
}

// validateIntKind check whether k is int kind
func validateIntKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

// validateUintKind check whether k is uint kind
func validateUintKind(k reflect.Kind) bool {
	switch k {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

// validateFloatKind check whether k is float kind
func validateFloatKind(k reflect.Kind) bool {
	switch k {
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// PackInt8 packs int to byte array
func PackInt8(v int8, b []byte) []byte {
	return append(b, byte(v))
}

// PackInt16 packs int16 to byte array
//[10].pack('N').bytes => [0, 0, 0, 10]
func PackInt16(v int16) []byte {
	var array [2]byte
	binary.BigEndian.PutUint16(array[:2], uint16(v))
	return array[:]
}

// PackUint16 packs uint16 to byte array
//[10].pack('N').bytes => [0, 0, 0, 10]
func PackUint16(v uint16) []byte {
	var array [2]byte
	binary.BigEndian.PutUint16(array[:2], v)
	return array[:]
}

// PackInt32 packs int32 to byte array
//[10].pack('N').bytes => [0, 0, 0, 10]
func PackInt32(v int32) []byte {
	var array [4]byte
	binary.BigEndian.PutUint32(array[:4], uint32(v))
	return array[:]
}

// PackInt64 packs int64 to byte array
//[10].pack('q>').bytes => [0, 0, 0, 0, 0, 0, 0, 10]
func PackInt64(v int64) []byte {
	var array [8]byte
	binary.BigEndian.PutUint64(array[:8], uint64(v))
	return array[:]
}

// PackFloat64 packs float64 to byte array
//[10].pack('G').bytes => [64, 36, 0, 0, 0, 0, 0, 0]
// PackFloat64 invokes go's official math library function Float64bits.
func PackFloat64(v float64) []byte {
	var array [8]byte
	binary.BigEndian.PutUint64(array[:8], math.Float64bits(v))
	return array[:]
}

// UnpackInt16 unpacks int16 from byte array
//(0,2).unpack('n')
func UnpackInt16(b []byte) int16 {
	arr := b[:2]
	return int16(binary.BigEndian.Uint16(arr))
}

// UnpackUint16 unpacks int16 from byte array
//(0,2).unpack('n')
func UnpackUint16(b []byte) uint16 {
	arr := b[:2]
	return binary.BigEndian.Uint16(arr)
}

// UnpackInt32 unpacks int32 from byte array
//(0,4).unpack('N')
func UnpackInt32(b []byte) int32 {
	arr := b[:4]
	return int32(binary.BigEndian.Uint32(arr))
}

// UnpackInt64 unpacks int64 from byte array
// long (0,8).unpack('q>')
func UnpackInt64(b []byte) int64 {
	arr := b[:8]
	return int64(binary.BigEndian.Uint64(arr))
}

// UnpackFloat64 unpacks float64 from byte array
// Double (0,8).unpack('G)
func UnpackFloat64(b []byte) float64 {
	arr := b[:8]
	return math.Float64frombits(binary.BigEndian.Uint64(arr))
}

// UnpackPtr unpack pointer value to original value
func UnpackPtr(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	return v
}

// PackPtr pack a Ptr value
func PackPtr(v reflect.Value) reflect.Value {
	vv := reflect.New(v.Type())
	vv.Elem().Set(v)
	return vv
}

// UnpackPtrType unpack pointer type to original type
func UnpackPtrType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ
}

// UnpackPtrValue unpack pointer value to original value
// return the pointer if its elem is zero value, because lots of operations on zero value is invalid
func UnpackPtrValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr && v.Elem().IsValid() {
		v = v.Elem()
	}
	return v
}

// SprintHex converts the []byte to a Hex string.
func SprintHex(b []byte) (rs string) {
	rs = fmt.Sprintf("[]byte{")
	for _, v := range b {
		rs += fmt.Sprintf("0x%02x,", v)
	}
	rs = strings.TrimSpace(rs)
	rs += fmt.Sprintf("}\n")
	return
}

// EnsurePackValue pack the interface with value
func EnsurePackValue(in interface{}) reflect.Value {
	if v, ok := in.(reflect.Value); ok {
		return v
	}
	return reflect.ValueOf(in)
}

// EnsureInterface get value of reflect.Value
// return original value if not reflect.Value
func EnsureInterface(in interface{}, err error) (interface{}, error) {
	if err != nil {
		return in, err
	}

	return EnsureRawAny(in), nil
}

// EnsureRawValue pack the interface with value, and make sure it's not a ref holder
func EnsureRawValue(in interface{}) reflect.Value {
	if v, ok := in.(reflect.Value); ok {
		if v.IsValid() {
			if r, ok := v.Interface().(*_refHolder); ok {
				return r.value
			}
		}
		return v
	}
	if v, ok := in.(*_refHolder); ok {
		return v.value
	}
	return reflect.ValueOf(in)
}

// EnsureRawAny unpack if in is a reflect.Value or a ref holder.
func EnsureRawAny(in interface{}) interface{} {
	if v, ok := in.(reflect.Value); ok {
		if !v.IsValid() {
			return nil
		}

		in = v.Interface()
	}

	if v, ok := in.(*_refHolder); ok {
		in = v.value
	}

	if v, ok := in.(reflect.Value); ok {
		if !v.IsValid() {
			return nil
		}

		in = v.Interface()
	}

	return in
}

// SetValue set the value to dest.
// It will auto check the Ptr pack level and unpack/pack to the right level.
// It make sure success to set value
func SetValue(dest, v reflect.Value) {
	// check whether the v is a ref holder
	if v.IsValid() {
		if h, ok := v.Interface().(*_refHolder); ok {
			h.add(dest)
			return
		}
	}
	// temporary process, only handle the same type of situation
	if v.IsValid() && UnpackPtrType(dest.Type()) == UnpackPtrType(v.Type()) && dest.Kind() == reflect.Ptr && dest.CanSet() {
		for dest.Type() != v.Type() {
			v = PackPtr(v)
		}
		dest.Set(v)
		return
	}

	// if the kind of dest is Ptr, the original value will be zero value
	// set value on zero value is not allowed
	// unpack to one-level pointer
	for dest.Kind() == reflect.Ptr && dest.Elem().Kind() == reflect.Ptr {
		dest = dest.Elem()
	}

	// if the kind of dest is Ptr, change the v to a Ptr value too.
	if dest.Kind() == reflect.Ptr {

		// unpack to one-level pointer
		for v.IsValid() && v.Kind() == reflect.Ptr && v.Elem().Kind() == reflect.Ptr {
			v = v.Elem()
		}
		// zero value not need to set
		if !v.IsValid() {
			return
		}

		if v.Kind() != reflect.Ptr {
			// change the v to a Ptr value
			v = PackPtr(v)
		}
	} else {
		v = UnpackPtrValue(v)
	}
	// zero value not need to set
	if !v.IsValid() {
		return
	}

	// set value as required type
	if dest.Type() == v.Type() && dest.CanSet() {
		dest.Set(v)
		return
	}

	// unpack ptr so that to special check for float,int,uint kind
	if dest.Kind() == reflect.Ptr {
		dest = UnpackPtrValue(dest)
		v = UnpackPtrValue(v)
	}

	kind := dest.Kind()
	switch kind {
	case reflect.Float32, reflect.Float64:
		dest.SetFloat(v.Float())
		return
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dest.SetInt(v.Int())
		return
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		dest.SetUint(v.Uint())
		return
	}

	dest.Set(v)
}

// AddrEqual compares addrs
func AddrEqual(x, y interface{}) bool {
	if x == nil || y == nil {
		return x == y
	}
	v1 := reflect.ValueOf(x)
	v2 := reflect.ValueOf(y)
	if v1.Type() != v2.Type() {
		return false
	}

	if v1.Kind() != reflect.Ptr {
		v1 = PackPtr(v1)
		v2 = PackPtr(v2)
	}
	return v1.Pointer() == v2.Pointer()
}

// SetSlice set value into slice object
func SetSlice(dest reflect.Value, objects interface{}) error {
	if objects == nil {
		return nil
	}

	dest = UnpackPtrValue(dest)
	destTyp := UnpackPtrType(dest.Type())
	elemKind := destTyp.Elem().Kind()
	if elemKind == reflect.Uint8 {
		// for binary
		dest.Set(EnsureRawValue(objects))
		return nil
	}

	if ref, ok := objects.(*_refHolder); ok {
		return unpackRefHolder(dest, destTyp, ref)
	}

	v := EnsurePackValue(objects)
	if h, ok := v.Interface().(*_refHolder); ok {
		// if the object is a ref one, just add the destination list to wait delay initialization
		h.add(dest)
		return nil
	}

	v, err := ConvertSliceValueType(destTyp, v)
	if err != nil {
		return err
	}
	SetValue(dest, v)
	return nil
}

// unpackRefHolder unpack the ref holder when decoding slice finished.
func unpackRefHolder(dest reflect.Value, destTyp reflect.Type, ref *_refHolder) error {
	v, err := ConvertSliceValueType(destTyp, ref.value)
	if err != nil {
		return err
	}
	SetValue(dest, v)
	ref.change(v) // change finally
	ref.notify()  // delay set value to all destinations
	return nil
}

// ConvertSliceValueType convert to slice of destination type
func ConvertSliceValueType(destTyp reflect.Type, v reflect.Value) (reflect.Value, error) {
	if destTyp == v.Type() || destTyp.Kind() == reflect.Interface {
		return v, nil
	}

	k := v.Type().Kind()
	if k != reflect.Slice && k != reflect.Array {
		return _zeroValue, perrors.Errorf("expect slice type, but get %v, objects: %v", k, v)
	}

	if v.Len() <= 0 {
		return _zeroValue, nil
	}

	elemKind := destTyp.Elem().Kind()
	elemPtrType := elemKind == reflect.Ptr
	elemFloatType := validateFloatKind(elemKind)
	elemIntType := validateIntKind(elemKind)
	elemUintType := validateUintKind(elemKind)

	sl := reflect.MakeSlice(destTyp, v.Len(), v.Len())
	var itemValue reflect.Value
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		if cv, ok := item.(reflect.Value); ok {
			itemValue = cv
		} else {
			if item == nil {
				itemValue = reflect.Zero(destTyp.Elem())
			} else {
				itemValue = reflect.ValueOf(item)
			}
		}

		if !elemPtrType && itemValue.Kind() == reflect.Ptr {
			itemValue = UnpackPtrValue(itemValue)
		}

		switch {
		case elemFloatType:
			sl.Index(i).SetFloat(itemValue.Float())
		case elemIntType:
			sl.Index(i).SetInt(itemValue.Int())
		case elemUintType:
			sl.Index(i).SetUint(itemValue.Uint())
		default:
			SetValue(sl.Index(i), itemValue)
		}
	}

	return sl, nil
}

//PackPtrInterface pack struct interface to pointer interface
func PackPtrInterface(s interface{}, value reflect.Value) interface{} {
	vv := reflect.New(reflect.TypeOf(s))
	vv.Elem().Set(value)
	return vv.Interface()
}
