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

// variable.Variable
// Deprecated: use mosn.io/pkg/variable/var.go:BasicVariable instead
type BasicVariable = variable.BasicVariable

// variable.Variable
// variable.VariableIndexer
// Deprecated: use mosn.io/pkg/variable/var.go:IndexedVariable instead
type IndexedVariable = variable.IndexedVariable

// NewBasicVariable
// Deprecated: use mosn.io/pkg/variable/var.go:NewBasicVariable instead
var NewBasicVariable = variable.NewBasicVariable

// NewIndexedVariable
// Deprecated: use mosn.io/pkg/variable/var.go:NewIndexedVariable instead
var NewIndexedVariable = variable.NewIndexedVariable

// BasicSetter used for variable value setting only, and would not affect any real data structure, like headers.
// Deprecated: use mosn.io/pkg/variable/var.go:BasicSetter instead
var BasicSetter = variable.BasicSetter
