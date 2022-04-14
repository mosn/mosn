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

	"github.com/stretchr/testify/require"
)

func TestNewVariableContext(t *testing.T) {
	testName := "test_variable"
	Register(NewStringVariable(testName, nil, nil, DefaultStringSetter, 0))
	ctx := NewVariableContext(context.Background())
	value := "test_value"
	require.Nil(t, SetString(ctx, testName, value))
	v, err := GetString(ctx, testName)
	require.Nil(t, err)
	require.Equal(t, value, v)
	// duplicate new varibale, will be ok
	ctx = NewVariableContext(ctx)
	v, err = GetString(ctx, testName)
	require.Nil(t, err)
	require.Equal(t, value, v)

}
