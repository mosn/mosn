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
	"mosn.io/pkg/variable"
)

const (
	MOSN_VAR_FLAG_CHANGEABLE  = variable.MOSN_VAR_FLAG_CHANGEABLE
	MOSN_VAR_FLAG_NOCACHEABLE = variable.MOSN_VAR_FLAG_NOCACHEABLE
	MOSN_VAR_FLAG_NOHASH      = variable.MOSN_VAR_FLAG_NOHASH

	ValueNotFound = variable.ValueNotFound
)

// GetterFunc used to get the value of variable, the implementation should handle the field
// (Valid, NotFound) of IndexedValue if it was not nil, Valid means the value is valid; NotFound
// means the value can not be found. It indicates that value can be cached for next-time get handle
// if any one of (Valid, NotFound) is set to true.
//
// Function should return ValueNotFound("-") if target value not exists.
// E.g. reference to the header which is not existed in current request.
// Deprecated: use mosn.io/pkg/variable/types.go:GetterFunc instead
type GetterFunc = variable.GetterFunc

// SetterFunc used to set the value of variable
// Deprecated: use mosn.io/pkg/variable/types.go:SetterFunc instead
type SetterFunc = variable.SetterFunc

// Variable provides a flexible and convenient way to pass information
// Deprecated: use mosn.io/pkg/variable/types.go:Variable instead
type Variable = variable.Variable

// IndexedValue used to store result value
// Deprecated: use mosn.io/pkg/variable/types.go:IndexedValue instead
type IndexedValue = variable.IndexedValue

// Indexer indicates that variable needs to be cached by using pre-allocated IndexedValue
// Deprecated: use mosn.io/pkg/variable/types.go:Indexer instead
type Indexer = variable.Indexer
