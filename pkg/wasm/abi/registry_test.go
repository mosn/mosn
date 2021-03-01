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
