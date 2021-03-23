package proxywasm

import "github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/rawhostcall"

func SetEffectiveContext(contextID uint32) {
	rawhostcall.ProxySetEffectiveContext(contextID)
}

func FinishVMContext() {
	rawhostcall.ProxyDone()
}
