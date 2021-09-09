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
	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

func TestEngine(t *testing.T) {
	var mockProto = &mockProtocol{}
	err := RegisterProtocol("engine-proto", mockProto)
	assert.Nil(t, err)

	err = RegisterMatcher("engine-proto", mockMatcher)
	assert.Nil(t, err)

	matcher := GetMatcher("engine-proto")
	assert.NotNil(t, matcher)

	var protocols = []string{"engine-proto"}
	xEngine, err := NewXEngine(protocols)
	assert.Nil(t, err)
	assert.NotNil(t, xEngine)

	_, res := xEngine.Match(context.TODO(), buffer.NewIoBuffer(10))
	assert.Equal(t, res, api.MatchSuccess)
}
