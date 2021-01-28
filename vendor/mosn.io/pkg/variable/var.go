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

import "context"

// variable.Variable
type BasicVariable struct {
	getter GetterFunc
	setter SetterFunc

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

func (bv *BasicVariable) Setter() SetterFunc {
	return bv.setter
}

func (bv *BasicVariable) Getter() GetterFunc {
	return bv.getter
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

// NewBasicVariable
func NewBasicVariable(name string, data interface{}, getter GetterFunc, setter SetterFunc, flags uint32) Variable {
	return &BasicVariable{
		getter: getter,
		setter: setter,
		name:   name,
		data:   data,
		flags:  flags,
	}
}

// NewIndexedVariable
func NewIndexedVariable(name string, data interface{}, getter GetterFunc, setter SetterFunc, flags uint32) Variable {
	return &IndexedVariable{
		BasicVariable: BasicVariable{
			getter: getter,
			setter: setter,
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
