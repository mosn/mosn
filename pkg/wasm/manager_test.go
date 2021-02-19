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

package wasm

import (
	"io/ioutil"
	"mosn.io/mosn/pkg/types"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mock"
)

func TestWasmManagerBasic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().Start().AnyTimes().Return(nil)
	module := mock.NewMockWasmModule(ctrl)
	module.EXPECT().NewInstance().AnyTimes().Return(instance)
	engine := mock.NewMockWasmVM(ctrl)
	engine.EXPECT().NewModule(gomock.Any()).AnyTimes().Return(module)

	RegisterWasmEngine("testWasmEngine", engine)

	_ = ioutil.WriteFile("/tmp/foo.wasm", []byte("some bytes"), 0644)

	config := v2.WasmPluginConfig{
		PluginName: "testPluginBasic",
		VmConfig: &v2.WasmVmConfig{
			Engine: "testWasmEngine",
			Path:   "/tmp/foo.wasm",
		},
		InstanceNum: 4,
	}

	// add config
	assert.Nil(t, GetWasmManager().AddOrUpdateWasm(config))

	// get config
	pw := GetWasmManager().GetWasmPluginWrapperByName("testPluginBasic")
	assert.NotNil(t, pw)
	assert.Equal(t, pw.GetPlugin().InstanceNum(), config.InstanceNum)

	assert.Nil(t, GetWasmManager().GetWasmPluginWrapperByName("pluginNotExists"))

	// uninstall non-existing config
	assert.Equal(t, GetWasmManager().UninstallWasmPluginByName("pluginNotExists"), ErrPluginNotFound)

	// uninstall config
	assert.Nil(t, GetWasmManager().UninstallWasmPluginByName("testPluginBasic"))

	// re-add config
	assert.Nil(t, GetWasmManager().AddOrUpdateWasm(config))

	// add same config
	config2 := config.Clone()
	assert.Equal(t, GetWasmManager().AddOrUpdateWasm(config2), ErrSameWasmConfig)
}

func TestWasmManagerDiffVmConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	newModuleCount, newInstanceCount := 0, 0
	module := mock.NewMockWasmModule(ctrl)
	module.EXPECT().NewInstance().AnyTimes().DoAndReturn(func() types.WasmInstance {
		newInstanceCount++
		instance := mock.NewMockWasmInstance(ctrl)
		instance.EXPECT().Start().AnyTimes().Return(nil)
		return instance
	})
	engine := mock.NewMockWasmVM(ctrl)
	engine.EXPECT().NewModule(gomock.Any()).AnyTimes().DoAndReturn(func([]byte) types.WasmModule {
		newModuleCount++
		return module
	})

	RegisterWasmEngine("testWasmEngine", engine)

	_ = ioutil.WriteFile("/tmp/foo.wasm", []byte("some bytes"), 0644)
	_ = ioutil.WriteFile("/tmp/foo2.wasm", []byte("some bytes"), 0644)

	config := v2.WasmPluginConfig{
		PluginName: "testPluginDiffWasm",
		VmConfig: &v2.WasmVmConfig{
			Engine: "testWasmEngine",
			Path:   "/tmp/foo.wasm",
		},
		InstanceNum: 4,
	}

	assert.Nil(t, GetWasmManager().AddOrUpdateWasm(config))

	config2 := config.Clone()
	config2.VmConfig.Path = "/tmp/foo2.wasm"

	assert.Nil(t, GetWasmManager().AddOrUpdateWasm(config2))
	assert.Equal(t, newModuleCount, 2)
	assert.Equal(t, newInstanceCount, 2*config.InstanceNum)
}

func TestWasmManagerInstanceNum(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	newInstanceCount := 0
	module := mock.NewMockWasmModule(ctrl)
	module.EXPECT().NewInstance().AnyTimes().DoAndReturn(func() types.WasmInstance {
		newInstanceCount++
		instance := mock.NewMockWasmInstance(ctrl)
		instance.EXPECT().Start().AnyTimes().Return(nil)
		return instance
	})
	engine := mock.NewMockWasmVM(ctrl)
	engine.EXPECT().NewModule(gomock.Any()).AnyTimes().Return(module)

	RegisterWasmEngine("testWasmEngine", engine)

	_ = ioutil.WriteFile("/tmp/foo.wasm", []byte("some bytes"), 0644)

	config := v2.WasmPluginConfig{
		PluginName: "testPluginInstanceNum",
		VmConfig: &v2.WasmVmConfig{
			Engine: "testWasmEngine",
			Path:   "/tmp/foo.wasm",
		},
		InstanceNum: 4,
	}
	assert.Nil(t, GetWasmManager().AddOrUpdateWasm(config))
	assert.Equal(t, newInstanceCount, 4)

	// invalid num
	config2 := config.Clone()
	config2.InstanceNum = 0
	assert.Nil(t, GetWasmManager().AddOrUpdateWasm(config2))
	assert.Equal(t, GetWasmManager().GetWasmPluginWrapperByName("testPluginInstanceNum").GetPlugin().InstanceNum(), 4)
	assert.Equal(t, newInstanceCount, 4)

	// shrink
	config3 := config2.Clone()
	config3.InstanceNum = 2
	assert.Nil(t, GetWasmManager().AddOrUpdateWasm(config3))
	assert.Equal(t, GetWasmManager().GetWasmPluginWrapperByName("testPluginInstanceNum").GetPlugin().InstanceNum(), 2)
	assert.Equal(t, newInstanceCount, 4)

	// expand
	config4 := config3.Clone()
	config4.InstanceNum = 10
	assert.Nil(t, GetWasmManager().AddOrUpdateWasm(config4))
	assert.Equal(t, GetWasmManager().GetWasmPluginWrapperByName("testPluginInstanceNum").GetPlugin().InstanceNum(), 10)
	assert.Equal(t, newInstanceCount, 4+8) // should create 8 new instances since we have shrinked to 2
}

func TestWasmEngine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	engine := mock.NewMockWasmVM(ctrl)

	RegisterWasmEngine("", engine)
	assert.Nil(t, GetWasmEngine(""))

	RegisterWasmEngine("testWasmEngineNormal", engine)
	assert.Equal(t, GetWasmEngine("testWasmEngineNormal"), engine)
}
