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

package variable

import (
	"context"
	"errors"
)

// variable.Variable
type BasicVariable struct {
	getter *getterImpl
	setter *setterImpl

	name  string
	data  interface{}
	flags uint32
}

func (bv *BasicVariable) Name() string {
	return bv.name
}

func (bv *BasicVariable) Data() interface{} {
	return bv.data
}

func (bv *BasicVariable) Flags() uint32 {
	return bv.flags
}

func (bv *BasicVariable) Getter() Getter {
	return bv.getter
}

func (bv *BasicVariable) Setter() Setter {
	return bv.setter
}

// variable.Variable
// variable.VariableIndexer
type IndexedVariable struct {
	BasicVariable

	index uint32
}

func (iv *IndexedVariable) SetIndex(index uint32) {
	iv.index = index
}

func (iv *IndexedVariable) GetIndex() uint32 {
	return iv.index
}

func NewStringVariable(name string, data interface{}, getter StringGetter, setter StringSetter, flags uint32) Variable {
	basic := BasicVariable{
		getter: &getterImpl{stringGetter: getter},
		setter: &setterImpl{stringSetter: setter},
		name:   name,
		data:   data,
		flags:  flags,
	}

	if setter != nil {
		return &IndexedVariable{BasicVariable: basic}
	}

	return &basic
}

func NewVariable(name string, data interface{}, getter InterfaceGetter, setter InterfaceSetter, flags uint32) Variable {
	basic := BasicVariable{
		getter: &getterImpl{interfaceGetter: getter},
		setter: &setterImpl{interfaceSetter: setter},
		name:   name,
		data:   data,
		flags:  flags,
	}

	if setter != nil {
		return &IndexedVariable{BasicVariable: basic}
	}

	return &basic
}

// Deprecated: use NewStringVariable instead.
func NewBasicVariable(name string, data interface{}, getter StringGetter, setter StringSetter, flags uint32) Variable {
	return &BasicVariable{
		getter: &getterImpl{stringGetter: getter},
		setter: &setterImpl{stringSetter: setter},
		name:   name,
		data:   data,
		flags:  flags,
	}
}

// Deprecated: use NewStringVariable instead.
func NewIndexedVariable(name string, data interface{}, getter StringGetter, setter StringSetter, flags uint32) Variable {
	return &IndexedVariable{
		BasicVariable: BasicVariable{
			getter: &getterImpl{stringGetter: getter},
			setter: &setterImpl{stringSetter: setter},
			name:   name,
			data:   data,
			flags:  flags,
		},
	}
}

// BasicSetter used for variable value setting only, and would not affect any real data structure, like headers.
func BasicSetter(ctx context.Context, variableValue *IndexedValue, value string) error {
	variableValue.data = value
	variableValue.Valid = true
	return nil
}

func BasicInterfaceSetter(ctx context.Context, variableValue *IndexedValue, value interface{}) error {
	variableValue.data = value
	variableValue.Valid = true
	return nil
}

type setterImpl struct {
	stringSetter    StringSetter
	interfaceSetter InterfaceSetter
}

func (s *setterImpl) Set(ctx context.Context, variableValue *IndexedValue, value interface{}) error {
	if s.stringSetter != nil {
		if v, ok := value.(string); ok {
			return s.stringSetter(ctx, variableValue, v)
		}
		return errors.New(errValueNotString)
	}

	if s.interfaceSetter != nil {
		return s.interfaceSetter(ctx, variableValue, value)
	}

	return errors.New(errSetterNotFound)
}

type getterImpl struct {
	stringGetter    StringGetter
	interfaceGetter InterfaceGetter
}

func (g *getterImpl) Get(ctx context.Context, value *IndexedValue, data interface{}) (interface{}, error) {
	if g.stringGetter != nil {
		return g.stringGetter(ctx, value, data)
	}

	if g.interfaceGetter != nil {
		return g.interfaceGetter(ctx, value, data)
	}

	return nil, errors.New(errGetterNotFound)
}
