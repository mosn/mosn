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

package protocol

import (
	"context"
	"testing"

	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
)

func TestMapping(t *testing.T) {
	if _, err := MappingHeaderStatusCode(context.Background(), Xprotocol, nil); err != ErrNoMapping {
		t.Error("no register type")
	}

	variable.RegisterVariable(variable.NewIndexedVariable(types.VarHeaderStatus, nil, nil, variable.BasicSetter, 0))
	ctx1 := variable.NewVariableContext(context.Background())

	variable.SetVariableValue(ctx1, types.VarHeaderStatus, "200")
	ctx2 := variable.NewVariableContext(context.Background())
	testcases := []struct {
		ctx      context.Context
		Expetced int
	}{
		{
			ctx1,
			200,
		},
		{
			ctx2,
			0,
		},
	}
	for i, tc := range testcases {
		code, _ := MappingHeaderStatusCode(tc.ctx, HTTP1, nil)
		if code != tc.Expetced {
			t.Errorf("#%d unexpected status code", i)
		}
	}

	for i, tc := range testcases {
		code, _ := MappingHeaderStatusCode(tc.ctx, HTTP2, nil)
		if code != tc.Expetced {
			t.Errorf("#%d unexpected status code", i)
		}
	}
}
