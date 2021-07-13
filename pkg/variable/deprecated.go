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

// Deprecated: AddVariable is used to check variable name exists. Typical usage is variables used in access logs.
// use Check instead
func AddVariable(name string) (Variable, error) {
	return Check(name)
}

// Deprecated: use Register instead
func RegisterVariable(variable Variable) error {
	return Register(variable)
}

// Deprecated: use RegisterPrefix instead
func RegisterPrefixVariable(prefix string, variable Variable) error {
	return RegisterPrefix(prefix, variable)
}

// Deprecated: use Get or GetString instead
func GetVariableValue(ctx context.Context, name string) (string, error) {
	return GetString(ctx, name)
}

// Deprecated: use Set or SetString instead
func SetVariableValue(ctx context.Context, name, value string) error {
	return SetString(ctx, name, value)
}

// Deprecated: use NewStringVariable instead.
func NewBasicVariable(name string, data interface{}, getter StringGetterFunc, setter StringSetterFunc, flags uint32) Variable {
	return NewStringVariable(name, data, getter, setter, flags)
}

// Deprecated: use NewStringVariable instead.
func NewIndexedVariable(name string, data interface{}, getter StringGetterFunc, setter StringSetterFunc, flags uint32) Variable {
	return NewStringVariable(name, data, getter, setter, flags)
}

// Deprecated: use DefaultStringSetter or DefaultSetter instead
func BasicSetter(ctx context.Context, variableValue *IndexedValue, value string) error {
	return DefaultStringSetter(ctx, variableValue, value)
}
