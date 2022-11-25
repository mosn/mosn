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

package flowcontrol

import (
	"context"
	"testing"

	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"

	"github.com/stretchr/testify/assert"
)

func TestDefaultCallbacks(t *testing.T) {
	mockConfig := &Config{
		GlobalSwitch: false,
		Monitor:      false,
		KeyType:      "PATH",
	}
	cb := &DefaultCallbacks{config: mockConfig}
	assert.False(t, cb.Enabled())
	mockConfig.GlobalSwitch = true
	assert.True(t, cb.Enabled())
}

func TestCallbacksRegistry(t *testing.T) {
	cfg := defaultConfig()
	cb := &DefaultCallbacks{config: defaultConfig()}
	RegisterCallbacks("default", cb)
	assert.NotNil(t, GetCallbacksByConfig(cfg))
	cfg.CallbackName = "default"
	assert.NotNil(t, GetCallbacksByConfig(cfg))
}

func TestAfterBlock(t *testing.T) {
	cfg := defaultConfig()
	filter := MockInboundFilter(cfg)
	cb := GetCallbacksByConfig(cfg)
	ctx := context.Background()
	ctx = variable.NewVariableContext(ctx)
	variable.Register(variable.NewStringVariable(types.VarHeaderStatus, nil, nil, variable.DefaultStringSetter, 0))
	ctx = variable.NewVariableContext(context.Background())
	variable.SetString(ctx, types.VarHeaderStatus, "200")

	cb.AfterBlock(filter, ctx, nil, nil, nil)
	status, err := variable.GetString(ctx, types.VarHeaderStatus)
	assert.Nil(t, err)
	assert.Equal(t, "509", status)
}

func TestDefaultCallbacks_SetConfig(t *testing.T) {
	cfg := defaultConfig()
	cb := GetCallbacksByConfig(cfg)
	assert.Empty(t, cb.GetConfig().CallbackName)
	cfg.CallbackName = "testing"
	cb = GetCallbacksByConfig(cfg)
	assert.Equal(t, "testing", cb.GetConfig().CallbackName)

	config := cb.GetConfig()
	assert.NotNil(t, config)
}
