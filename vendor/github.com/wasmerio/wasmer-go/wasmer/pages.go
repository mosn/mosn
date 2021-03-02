package wasmer

// #include <wasmer_wasm.h>
import "C"

// Units of WebAssembly pages (as specified to be 65,536 bytes).
type Pages C.wasm_memory_pages_t

// Represents a memory page size.
const WasmPageSize = uint(0x10000)

// Represents the maximum number of pages.
const WasmMaxPages = uint(0x10000)

// Represents the minimum number of pages.
const WasmMinPages = uint(0x100)

// ToUint32 converts a Pages to a native Go uint32 which is the Pages' size.
//
//   memory, _ := instance.Exports.GetMemory("exported_memory")
//   size := memory.Size().ToUint32()
//
func (self *Pages) ToUint32() uint32 {
	return uint32(C.wasm_memory_pages_t(*self))
}

// ToBytes converts a Pages to a native Go uint which is the Pages' size in bytes.
//
//   memory, _ := instance.Exports.GetMemory("exported_memory")
//   size := memory.Size().ToBytes()
//
func (self *Pages) ToBytes() uint {
	return uint(self.ToUint32()) * WasmPageSize
}
