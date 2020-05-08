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

package gzip

import (
	"context"
	"testing"

	"mosn.io/mosn/pkg/config/v2"
	_ "mosn.io/mosn/pkg/proxy"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
)

func TestGzipNewStreamFilter(t *testing.T) {
	// test disable
	cfg := &v2.StreamGzip{}
	ctx := variable.NewVariableContext(context.Background())
	NewStreamFilter(ctx, cfg)

	// check gzip switch
	if gzipSwitch, err := variable.GetVariableValue(ctx, types.VarProxyGzipSwitch); err != nil || gzipSwitch != "off" {
		t.Error("gzip should be disable.")
	}

	// test enable gzip
	cfg = &v2.StreamGzip{
		GzipLevel: 2,
	}
	ctx = variable.NewVariableContext(context.Background())
	NewStreamFilter(ctx, cfg)
	if gzipSwitch, err := variable.GetVariableValue(ctx, types.VarProxyGzipSwitch); err != nil || gzipSwitch != "on" {
		t.Error("gzip should be enable.")
	}
}
