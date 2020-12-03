// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"

	structpb "github.com/golang/protobuf/ptypes/struct"
)

// baseMap is a reflection based map implementation designed to handle a variety of map-like types.
type baseMap struct {
	ref.TypeAdapter
	value    interface{}
	refValue reflect.Value
}

// stringMap is a specialization to improve the performance of simple key, value pair lookups by
// string as this is the most common usage of maps.
type stringMap struct {
	*baseMap
	mapStrStr map[string]string
}

// NewDynamicMap returns a traits.Mapper value with dynamic key, value pairs.
func NewDynamicMap(adapter ref.TypeAdapter, value interface{}) traits.Mapper {
	return &baseMap{
		TypeAdapter: adapter,
		value:       value,
		refValue:    reflect.ValueOf(value)}
}

// NewStringStringMap returns a specialized traits.Mapper with string keys and values.
func NewStringStringMap(adapter ref.TypeAdapter, value map[string]string) traits.Mapper {
	return &stringMap{
		baseMap:   &baseMap{TypeAdapter: adapter, value: value},
		mapStrStr: value,
	}
}

var (
	// MapType singleton.
	MapType = NewTypeValue("map",
		traits.ContainerType,
		traits.IndexerType,
		traits.IterableType,
		traits.SizerType)
)

// Contains implements the traits.Container interface method.
func (m *baseMap) Contains(index ref.Val) ref.Val {
	val, found := m.Find(index)
	// When the index is not found and val is non-nil, this is an error or unknown value.
	if !found && val != nil && IsUnknownOrError(val) {
		return val
	}
	return Bool(found)
}

// Contains implements the traits.Container interface method.
func (m *stringMap) Contains(index ref.Val) ref.Val {
	val, found := m.Find(index)
	// When the index is not found and val is non-nil, this is an error or unknown value.
	if !found && val != nil && IsUnknownOrError(val) {
		return val
	}
	return Bool(found)
}

// ConvertToNative implements the ref.Val interface method.
func (m *baseMap) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	switch typeDesc {
	case anyValueType:
		json, err := m.ConvertToNative(jsonStructType)
		if err != nil {
			return nil, err
		}
		return ptypes.MarshalAny(json.(proto.Message))
	case jsonValueType, jsonStructType:
		jsonEntries, err :=
			m.ConvertToNative(reflect.TypeOf(map[string]*structpb.Value{}))
		if err != nil {
			return nil, err
		}
		jsonMap := &structpb.Struct{
			Fields: jsonEntries.(map[string]*structpb.Value)}
		if typeDesc == jsonStructType {
			return jsonMap, nil
		}
		return &structpb.Value{
			Kind: &structpb.Value_StructValue{
				StructValue: jsonMap}}, nil
	}

	// Unwrap pointers, but track their use.
	isPtr := false
	if typeDesc.Kind() == reflect.Ptr {
		tk := typeDesc
		typeDesc = typeDesc.Elem()
		if typeDesc.Kind() == reflect.Ptr {
			return nil, fmt.Errorf("unsupported type conversion to '%v'", tk)
		}
		isPtr = true
	}

	// If the map is already assignable to the desired type return it, e.g. interfaces and
	// maps with the same key value types.
	if reflect.TypeOf(m).AssignableTo(typeDesc) {
		return m, nil
	}

	// Establish some basic facts about the map key and value types.
	thisType := m.refValue.Type()
	thisKey := thisType.Key()
	thisKeyKind := thisKey.Kind()
	thisElem := thisType.Elem()
	thisElemKind := thisElem.Kind()

	switch typeDesc.Kind() {
	// Map conversion.
	case reflect.Map:
		otherKey := typeDesc.Key()
		otherKeyKind := otherKey.Kind()
		otherElem := typeDesc.Elem()
		otherElemKind := otherElem.Kind()
		if otherKeyKind == thisKeyKind && otherElemKind == thisElemKind {
			return m.value, nil
		}
		elemCount := m.Size().(Int)
		nativeMap := reflect.MakeMapWithSize(typeDesc, int(elemCount))
		it := m.Iterator()
		for it.HasNext() == True {
			key := it.Next()
			refKeyValue, err := key.ConvertToNative(otherKey)
			if err != nil {
				return nil, err
			}
			refElemValue, err := m.Get(key).ConvertToNative(otherElem)
			if err != nil {
				return nil, err
			}
			nativeMap.SetMapIndex(
				reflect.ValueOf(refKeyValue),
				reflect.ValueOf(refElemValue))
		}
		return nativeMap.Interface(), nil
	case reflect.Struct:
		if thisKeyKind != reflect.String && thisKeyKind != reflect.Interface {
			break
		}
		nativeStructPtr := reflect.New(typeDesc)
		nativeStruct := nativeStructPtr.Elem()
		it := m.Iterator()
		for it.HasNext() == True {
			key := it.Next()
			// Ensure the field name being referenced is exported.
			// Only exported (public) field names can be set by reflection, where the name
			// must be at least one character in length and start with an upper-case letter.
			fieldName := string(key.ConvertToType(StringType).(String))
			switch len(fieldName) {
			case 0:
				return nil, errors.New("type conversion error, unsupported empty field")
			case 1:
				fieldName = strings.ToUpper(fieldName)
			default:
				fieldName = strings.ToUpper(fieldName[0:1]) + fieldName[1:]
			}
			fieldRef := nativeStruct.FieldByName(fieldName)
			if !fieldRef.IsValid() {
				return nil, fmt.Errorf(
					"type conversion error, no such field '%s' in type '%v'",
					fieldName, typeDesc)
			}
			fieldValue, err := m.Get(key).ConvertToNative(fieldRef.Type())
			if err != nil {
				return nil, err
			}
			fieldRef.Set(reflect.ValueOf(fieldValue))
		}
		if isPtr {
			return nativeStructPtr.Interface(), nil
		}
		return nativeStruct.Interface(), nil
	}
	return nil, fmt.Errorf("type conversion error from map to '%v'", typeDesc)
}

// ConvertToNative implements the ref.Val interface method.
func (m *stringMap) ConvertToNative(refType reflect.Type) (interface{}, error) {
	if !m.baseMap.refValue.IsValid() {
		m.baseMap.refValue = reflect.ValueOf(m.value)
	}
	return m.baseMap.ConvertToNative(refType)
}

// ConvertToType implements the ref.Val interface method.
func (m *baseMap) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case MapType:
		return m
	case TypeType:
		return MapType
	}
	return NewErr("type conversion error from '%s' to '%s'", MapType, typeVal)
}

// ConvertToType implements the ref.Val interface method.
func (m *stringMap) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case MapType:
		return m
	default:
		return m.baseMap.ConvertToType(typeVal)
	}
}

// Equal implements the ref.Val interface method.
func (m *baseMap) Equal(other ref.Val) ref.Val {
	if MapType != other.Type() {
		return ValOrErr(other, "no such overload")
	}
	otherMap := other.(traits.Mapper)
	if m.Size() != otherMap.Size() {
		return False
	}
	it := m.Iterator()
	for it.HasNext() == True {
		key := it.Next()
		thisVal, _ := m.Find(key)
		otherVal, found := otherMap.Find(key)
		if !found {
			if otherVal == nil {
				return False
			}
			return ValOrErr(otherVal, "no such overload")
		}
		valEq := thisVal.Equal(otherVal)
		if valEq != True {
			return valEq
		}
	}
	return True
}

// Equal implements the ref.Val interface method.
func (m *stringMap) Equal(other ref.Val) ref.Val {
	if !m.baseMap.refValue.IsValid() {
		m.baseMap.refValue = reflect.ValueOf(m.value)
	}
	return m.baseMap.Equal(other)
}

// Find implements the traits.Mapper interface method.
func (m *baseMap) Find(key ref.Val) (ref.Val, bool) {
	// TODO: There are multiple reasons why a Get could fail. Typically, this is because the key
	// does not exist in the map; however, it's possible that the value cannot be converted to
	// the desired type. Refine this strategy to disambiguate these cases.
	if IsUnknownOrError(key) {
		return key, false
	}
	thisKeyType := m.refValue.Type().Key()
	nativeKey, err := key.ConvertToNative(thisKeyType)
	if err != nil {
		return &Err{err}, false
	}
	nativeKeyVal := reflect.ValueOf(nativeKey)
	value := m.refValue.MapIndex(nativeKeyVal)
	if !value.IsValid() {
		return nil, false
	}
	return m.NativeToValue(value.Interface()), true
}

// Find implements the traits.Mapper interface method.
func (m *stringMap) Find(key ref.Val) (ref.Val, bool) {
	strKey, ok := key.(String)
	if !ok {
		return ValOrErr(key, "no such overload"), false
	}
	val, found := m.mapStrStr[string(strKey)]
	if !found {
		return nil, false
	}
	return String(val), true
}

// Get implements the traits.Indexer interface method.
func (m *baseMap) Get(key ref.Val) ref.Val {
	v, found := m.Find(key)
	if !found {
		return ValOrErr(v, "no such key: %v", key)
	}
	return v
}

// Get implements the traits.Indexer interface method.
func (m *stringMap) Get(key ref.Val) ref.Val {
	v, found := m.Find(key)
	if !found {
		return ValOrErr(v, "no such key: %v", key)
	}
	return v
}

// Iterator implements the traits.Iterable interface method.
func (m *baseMap) Iterator() traits.Iterator {
	mapKeys := m.refValue.MapKeys()
	return &mapIterator{
		baseIterator: &baseIterator{},
		TypeAdapter:  m.TypeAdapter,
		mapValue:     m,
		mapKeys:      mapKeys,
		cursor:       0,
		len:          int(m.Size().(Int))}
}

// Iterator implements the traits.Iterable interface method.
func (m *stringMap) Iterator() traits.Iterator {
	if !m.baseMap.refValue.IsValid() {
		m.baseMap.refValue = reflect.ValueOf(m.value)
	}
	return m.baseMap.Iterator()
}

// Size implements the traits.Sizer interface method.
func (m *baseMap) Size() ref.Val {
	return Int(m.refValue.Len())
}

// Size implements the traits.Sizer interface method.
func (m *stringMap) Size() ref.Val {
	return Int(len(m.mapStrStr))
}

// Type implements the ref.Val interface method.
func (m *baseMap) Type() ref.Type {
	return MapType
}

// Value implements the ref.Val interface method.
func (m *baseMap) Value() interface{} {
	return m.value
}

type mapIterator struct {
	*baseIterator
	ref.TypeAdapter
	mapValue traits.Mapper
	mapKeys  []reflect.Value
	cursor   int
	len      int
}

// HasNext implements the traits.Iterator interface method.
func (it *mapIterator) HasNext() ref.Val {
	return Bool(it.cursor < it.len)
}

// Next implements the traits.Iterator interface method.
func (it *mapIterator) Next() ref.Val {
	if it.HasNext() == True {
		index := it.cursor
		it.cursor++
		refKey := it.mapKeys[index]
		return it.NativeToValue(refKey.Interface())
	}
	return nil
}
