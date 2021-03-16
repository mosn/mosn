package wasm

import (
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm/abi"
	"mosn.io/mosn/pkg/wasm/abi/ext/xproxywasm020"
	v1 "mosn.io/mosn/pkg/wasm/abi/proxywasm010"
)

func init() {
	abi.RegisterABI(xproxywasm020.AbiV2, abiImplFactory)
}

func abiImplFactory(instance types.WasmInstance) types.ABI {
	abi := &AbiV2Impl{}
	abi.SetImports(&protocolWrapper{})
	abi.SetInstance(instance)
	return abi
}

// easy for extension
type AbiV2Impl struct {
	v1.ABIContext
}

func (a *AbiV2Impl) Name() string {
	return xproxywasm020.AbiV2
}
