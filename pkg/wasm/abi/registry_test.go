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

package abi

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/mock"
	"mosn.io/mosn/pkg/types"
)

func TestRegistryGetABI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	RegisterABI("abi1", func(instance types.WasmInstance) types.ABI { return mock.NewMockABI(ctrl) })

	module := mock.NewMockWasmModule(ctrl)
	module.EXPECT().GetABINameList().AnyTimes().Return([]string{"abi1"})
	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetModule().AnyTimes().Return(module)

	ret := GetABI(instance, "abi1")
	assert.NotNil(t, ret)

	assert.Nil(t, GetABI(nil, "abi2"))
	assert.Nil(t, GetABI(instance, ""))
	assert.Nil(t, GetABI(instance, "otherABI"))
}

func TestRegistryGetABIList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	RegisterABI("abi1", func(instance types.WasmInstance) types.ABI { return mock.NewMockABI(ctrl) })
	RegisterABI("abi2", func(instance types.WasmInstance) types.ABI { return mock.NewMockABI(ctrl) })
	RegisterABI("abi3", func(instance types.WasmInstance) types.ABI { return mock.NewMockABI(ctrl) })

	module := mock.NewMockWasmModule(ctrl)
	module.EXPECT().GetABINameList().AnyTimes().Return([]string{"abi1", "abi2", "abi4"})
	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetModule().AnyTimes().Return(module)

	ret := GetABIList(instance)
	assert.NotNil(t, ret)
	assert.Equal(t, len(ret), 2)
}
