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
	MOSN_VAR_FLAG_CHANGEABLE  = 1
	MOSN_VAR_FLAG_NOCACHEABLE = 2
	MOSN_VAR_FLAG_NOHASH      = 4

	ValueNotFound = "-"
)

// GetterFunc used to get the value of variable, the implementation should handle the field
// (Valid, NotFound) of IndexedValue if it was not nil, Valid means the value is valid; NotFound
// means the value can not be found. It indicates that value can be cached for next-time get handle
// if any one of (Valid, NotFound) is set to true.
//
// Function should return ValueNotFound("-") if target value not exists.
// E.g. reference to the header which is not existed in current request.
type GetterFunc func(ctx context.Context, value *IndexedValue, data interface{}) (string, error)

// SetterFunc used to set the value of variable
type SetterFunc func(ctx context.Context, variableValue *IndexedValue, value string) error

// Variable provides a flexible and convenient way to pass information
type Variable interface {
	// variable name
	Name() string
	// variable data, which is useful for getter/setter
	Data() interface{}
	// variable flags
	Flags() uint32
	// value getter
	Getter() GetterFunc
	// value setter
	Setter() SetterFunc
}

// IndexedValue used to store result value
type IndexedValue struct {
	Valid       bool
	NotFound    bool
	noCacheable bool
	//escape      bool

	data string
}

// Indexer indicates that variable needs to be cached by using pre-allocated IndexedValue
type Indexer interface {
	// variable index
	GetIndex() uint32

	// set index to variable
	SetIndex(index uint32)
}
