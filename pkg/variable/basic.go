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

type BasicVariable struct {
	name  string
	data  interface{}
	flags uint32
	index uint32

	getter VariableGetter
	setter VariableSetter
}

func (httpVar *BasicVariable) Name() string {
	return httpVar.name
}

func (httpVar *BasicVariable) Data() interface{} {
	return httpVar.data
}

func (httpVar *BasicVariable) Flags() uint32 {
	return httpVar.flags
}

func (httpVar *BasicVariable) Setter() VariableSetter {
	return httpVar.setter
}

func (httpVar *BasicVariable) Getter() VariableGetter {
	return httpVar.getter
}

func (httpVar *BasicVariable) SetIndex(index uint32) {
	httpVar.index = index
}

func (httpVar *BasicVariable) GetIndex() uint32 {
	return httpVar.index
}

func NewSimpleBasicVariable(name string, getter VariableGetter, flags uint32) Variable {
	return &BasicVariable{
		name:   name,
		data:   name,
		getter: getter,
		flags:  flags,
	}
}

func NewBasicVariable(name string, data interface{}, getter VariableGetter, setter VariableSetter, flags uint32) Variable {
	return &BasicVariable{
		name:   name,
		data:   data,
		getter: getter,
		setter: setter,
		flags:  flags,
	}
}
