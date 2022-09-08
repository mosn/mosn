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

const (
	ValueNotFound = "-"
)

// StringGetterFunc used to get the value of string-typed variable, the implementation should handle the field
// Valid of IndexedValue if it was not nil, Valid means the value is valid.
// Function should return ValueNotFound("-") if target value not exists.
// E.g. reference to the header which is not existed in current request.
type StringGetterFunc func(ctx context.Context, value *IndexedValue, data interface{}) (string, error)

// GetterFunc used to get the value of interface-typed variable
type GetterFunc func(ctx context.Context, value *IndexedValue, data interface{}) (interface{}, error)

type Getter interface {
	Get(ctx context.Context, value *IndexedValue, data interface{}) (interface{}, error)
}

// StringSetterFunc used to set the value of string-typed variable
type StringSetterFunc func(ctx context.Context, variableValue *IndexedValue, value string) error

// SetterFunc used to set the value of interface-typed variable
type SetterFunc func(ctx context.Context, variableValue *IndexedValue, value interface{}) error

type Setter interface {
	Set(ctx context.Context, variableValue *IndexedValue, value interface{}) error
}

// Variable provides a flexible and convenient way to pass information
type Variable interface {
	// variable name
	Name() string
	// variable data, which is useful for getter/setter
	Data() interface{}
	// value getter
	Getter() Getter
	// value setter
	Setter() Setter
}

// IndexedValue used to store result value
type IndexedValue struct {
	Valid bool

	data interface{}
}

// Indexer indicates that variable needs to be cached by using pre-allocated IndexedValue
type Indexer interface {
	// variable index
	GetIndex() uint32

	// set index to variable
	SetIndex(index uint32)
}
