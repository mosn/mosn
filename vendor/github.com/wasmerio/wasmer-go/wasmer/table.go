package wasmer

// #include <wasmer_wasm.h>
import "C"
import "runtime"

// TableSize represents the size of a table.
type TableSize C.wasm_table_size_t

// ToUint32 converts a TableSize to a native Go uint32.
//
//   table, _ := instance.Exports.GetTable("exported_table")
//   size := table.Size().ToUint32()
func (self *TableSize) ToUint32() uint32 {
	return uint32(C.wasm_table_size_t(*self))
}

// A table instance is the runtime representation of a table. It holds
// a vector of function elements and an optional maximum size, if one
// was specified in the table type at the table’s definition site.
//
// See also
//
// Specification: https://webassembly.github.io/spec/core/exec/runtime.html#table-instances
type Table struct {
	_inner   *C.wasm_table_t
	_ownedBy interface{}
}

func newTable(pointer *C.wasm_table_t, ownedBy interface{}) *Table {
	table := &Table{_inner: pointer, _ownedBy: ownedBy}

	if ownedBy == nil {
		runtime.SetFinalizer(table, func(table *Table) {
			C.wasm_table_delete(table.inner())
		})
	}

	return table
}

func (self *Table) inner() *C.wasm_table_t {
	return self._inner
}

func (self *Table) ownedBy() interface{} {
	if self._ownedBy == nil {
		return self
	}

	return self._ownedBy
}

// Size returns the Table's size.
//
//   table, _ := instance.Exports.GetTable("exported_table")
//   size := table.Size()
//
func (self *Table) Size() TableSize {
	return TableSize(C.wasm_table_size(self.inner()))
}

// IntoExtern converts the Table into an Extern.
//
//   table, _ := instance.Exports.GetTable("exported_table")
//   extern := table.IntoExtern()
//
func (self *Table) IntoExtern() *Extern {
	pointer := C.wasm_table_as_extern(self.inner())

	return newExtern(pointer, self.ownedBy())
}
