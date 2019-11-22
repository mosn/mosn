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
	"errors"
	"sync"
	"context"
	"strings"
	"sofastack.io/sofa-mosn/pkg/types"
	mosnctx "sofastack.io/sofa-mosn/pkg/context"
)

var (
	// global scope
	mux              sync.RWMutex
	variables        = make(map[string]Variable, 32)       // all built-in variable definitions
	prefixGetters    = make(map[string]VariableGetter, 32) // all prefix getter definitions
	indexedVariables = make([]Variable, 0, 32)             // indexed variables

	// request scope
	indexedValues = make([]VariableValue, 0, 32) // indexed values, which means its' memory is pre-allocated

	// error message
	errVariableDuplicated = "duplicate variable register, name: "
	errPrefixDuplicated   = "duplicate prefix variable register, prefix: "
	errUndefinedVariable  = "undefined variable, name: "
)

// TODO: for runtime variable definition, e.g. variable from configs or codes
func AddVariable(name string) {

}

func RegisterVariable(variable Variable) error {
	mux.Lock()
	defer mux.Unlock()

	name := variable.Name()

	// check conflict
	if _, ok := variables[name]; ok {
		return errors.New(errVariableDuplicated + name)
	}

	// register
	variables[name] = variable

	// check index
	if indexer, ok := variable.(VariableIndexer); ok {
		index := len(indexedVariables)
		indexer.SetIndex(uint32(index))

		indexedVariables = append(indexedVariables, variable)
		indexedValues = append(indexedValues, VariableValue{})
	}
	return nil
}

func RegisterPrefixVariable(prefix string, getter VariableGetter) error {
	mux.Lock()
	defer mux.Unlock()

	// check conflict
	if _, ok := prefixGetters[prefix]; ok {
		return errors.New(errPrefixDuplicated + prefix)
	}

	// register
	prefixGetters[prefix] = getter
	return nil
}

func GetVariableValue(ctx context.Context, name string) (string, error) {
	// find built-in variables
	if variable, ok := variables[name]; ok {
		// check index
		if indexer, ok := variable.(VariableIndexer); ok {
			return getFlushedVariableValue(ctx, indexer.GetIndex()), nil
		}
		getter := variable.Getter()
		return getter(ctx, variable.Data()), nil
	}

	// check prefix getter
	for prefix, getter := range prefixGetters {
		if strings.HasPrefix(name, prefix) {
			return getter(ctx, name), nil
		}
	}

	return "", errors.New(errUndefinedVariable + name)
}

func NewVariableContext(ctx context.Context) context.Context {
	// TODO: sync.Pool reuse
	values := make([]VariableValue, len(indexedValues)) // *2 buffer for runtime variable
	//copy(values, indexedValues)

	return mosnctx.WithValue(ctx, types.ContextKeyVariables, values)
}

// TODO: provide direct access to this function, so the cost of variable name finding could be optimized
func getFlushedVariableValue(ctx context.Context, index uint32) string {
	if variables := ctx.Value(types.ContextKeyVariables); variables != nil {
		if values, ok := variables.([]VariableValue); ok {
			value := values[index]
			if value.valid || value.notFound {
				if !value.noCacheable {
					return value.data
				}

				// clear flags
				value.valid = false
				value.notFound = false
			}

			return getIndexedVariableValue(ctx, index)
		}
	}

	return "-"
}

func getIndexedVariableValue(ctx context.Context, index uint32) string {
	variable := indexedVariables[index]
	value := indexedValues[index]

	if value.notFound || value.valid {
		return value.data;
	}

	getter := variable.Getter()
	value.data = getter(ctx, variable.Data())

	if value.data != "" {
		if (variable.Flags() & MOSN_VAR_FLAG_NOCACHEABLE) == 1 {
			value.noCacheable = true
		}
		return value.data
	}

	value.valid = false
	value.notFound = true
	return "-"
}
