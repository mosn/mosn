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

package xprotocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/types"
)

// type xprotocolMapping struct{}

func TestMapping(t *testing.T) {
	var ctx = context.TODO()
	var xm = xprotocolMapping{}
	// 1, sub protocol is nil
	_, err := xm.MappingHeaderStatusCode(ctx, nil)
	assert.NotNil(t, err)

	// 2. cannot get mapping
	mCtx := mosnctx.WithValue(ctx, types.ContextSubProtocol, "xxx-proto")
	_, err = xm.MappingHeaderStatusCode(mCtx, nil)
	assert.NotNil(t, err)

	// 3. normal
}
