package wasmer

// #include <wasmer_wasm.h>
import "C"
import (
	"runtime"
)

func newName(str string) C.wasm_name_t {
	var name C.wasm_name_t

	C.wasm_name_new_from_string(&name, C.CString(str))

	runtime.KeepAlive(str)

	return name
}

func nameToString(name *C.wasm_name_t) string {
	if name.data == nil {
		return ""
	}

	return C.GoStringN(name.data, C.int(name.size))
}
