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
	getter *basicGetter
	setter *basicSetter

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

type Option func(variable *BasicVariable)

func WithInterfaceGetter(getter GetterFuncInterface) Option {
	return func(variable *BasicVariable) {
		if variable.getter.stringGetter == nil {
			variable.getter.interfaceGetter = getter
		}
	}
}

func WithInterfaceSetter(setter SetterFuncInterface) Option {
	return func(variable *BasicVariable) {
		if variable.setter.stringSetter == nil {
			variable.setter.interfaceSetter = setter
		}
	}
}

// NewBasicVariable
func NewBasicVariable(name string, data interface{}, getter GetterFunc, setter SetterFunc, flags uint32, options ...Option) Variable {
	v := &BasicVariable{
		getter: &basicGetter{stringGetter: getter},
		setter: &basicSetter{stringSetter: setter},
		name:   name,
		data:   data,
		flags:  flags,
	}

	for _, op := range options {
		op(v)
	}

	return v
}

// NewIndexedVariable
func NewIndexedVariable(name string, data interface{}, getter GetterFunc, setter SetterFunc, flags uint32, options ...Option) Variable {
	v := BasicVariable{
		getter: &basicGetter{stringGetter: getter},
		setter: &basicSetter{stringSetter: setter},
		name:   name,
		data:   data,
		flags:  flags,
	}

	for _, op := range options {
		op(&v)
	}

	return &IndexedVariable{BasicVariable: v}
}

// BasicSetter used for variable value setting only, and would not affect any real data structure, like headers.
func BasicSetter(ctx context.Context, variableValue *IndexedValue, value string) error {
	variableValue.data = value
	variableValue.Valid = true
	return nil
}

func BasicSetterInterface(ctx context.Context, variableValue *IndexedValue, value interface{}) error {
	variableValue.data = value
	variableValue.Valid = true
	return nil
}

type basicSetter struct {
	stringSetter    SetterFunc
	interfaceSetter SetterFuncInterface
}

func (b *basicSetter) Set(ctx context.Context, variableValue *IndexedValue, value string) error {
	return b.stringSetter(ctx, variableValue, value)
}

func (b *basicSetter) SetInterface(ctx context.Context, variableValue *IndexedValue, value interface{}) error {
	if b.stringSetter != nil {
		if s, ok := value.(string); ok {
			return b.stringSetter(ctx, variableValue, s)
		} else {
			return errors.New(errValueNotString)
		}
	} else if b.interfaceSetter != nil {
		return b.interfaceSetter(ctx, variableValue, value)
	}
	return errors.New(errSetterNotFound)
}

type basicGetter struct {
	stringGetter    GetterFunc
	interfaceGetter GetterFuncInterface
}

func (b *basicGetter) Get(ctx context.Context, value *IndexedValue, data interface{}) (string, error) {
	return b.stringGetter(ctx, value, data)
}

func (b *basicGetter) GetInterface(ctx context.Context, value *IndexedValue, data interface{}) (interface{}, error) {
	if b.stringGetter != nil {
		return b.stringGetter(ctx, value, data)
	} else if b.interfaceGetter != nil {
		return b.interfaceGetter(ctx, value, data)
	}
	return nil, errors.New(errGetterNotFound)
}
