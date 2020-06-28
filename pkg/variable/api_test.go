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
	"testing"
)

func TestGetVariableValue_normal(t *testing.T) {
	name := "ApiGet"
	value := "Getter Result"

	// register test variable
	RegisterVariable(NewBasicVariable(name, nil, func(ctx context.Context, variableValue *IndexedValue, data interface{}) (s string, err error) {
		return value, nil
	}, nil, 0))

	ctx := context.Background()
	ctx = NewVariableContext(ctx)

	vv, err := GetVariableValue(ctx, name)
	if err != nil {
		t.Error(err)
	}

	if vv != value {
		t.Errorf("get value not equal, expected: %s, acutal: %s", value, vv)
	}
}

func TestSetVariableValue_normal(t *testing.T) {
	name := "ApiSet"
	value := "Setter Value"

	// register test variable
	RegisterVariable(NewIndexedVariable(name, nil, nil, BasicSetter, 0))

	ctx := context.Background()
	ctx = NewVariableContext(ctx)

	err := SetVariableValue(ctx, name, value)
	if err != nil {
		t.Error(err)
	}

	vv, err := GetVariableValue(ctx, name)
	if err != nil {
		t.Error(err)
	}

	if vv != value {
		t.Errorf("get/set value not equal, expected: %s, acutal: %s", value, vv)
	}
}
