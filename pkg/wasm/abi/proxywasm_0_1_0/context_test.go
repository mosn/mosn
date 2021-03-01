package proxywasm_0_1_0

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/mock"
)

func TestOnInstanceCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := &AbiContext{}
	assert.Equal(t, ctx.Name(), ProxyWasmABI_0_1_0)

	imports := newMockImportsHandler()
	ctx.SetImports(imports)
	assert.Equal(t, ctx.imports, imports)

	instance := mock.NewMockWasmInstance(ctrl)
	ctx.SetInstance(instance)
	assert.Equal(t, ctx.instance, instance)

	registerCount := 0
	instance.EXPECT().RegisterFunc(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(namespace string, funcName string, f interface{}) error {
			registerCount++
			return nil
		})

	ctx.OnInstanceCreate(instance)
	ctx.OnInstanceStart(instance)
	ctx.OnInstanceDestroy(instance)

	assert.Equal(t, registerCount, 30)
}
