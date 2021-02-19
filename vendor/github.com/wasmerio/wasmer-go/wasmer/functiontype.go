package wasmer

// #include <wasmer_wasm.h>
import "C"
import "runtime"

// FunctionType classifies the signature of functions, mapping a vector of parameters to a vector of results. They are
// also used to classify the inputs and outputs of instructions.
//
// See also
//
// Specification: https://webassembly.github.io/spec/core/syntax/types.html#function-types
//
type FunctionType struct {
	_inner   *C.wasm_functype_t
	_ownedBy interface{}
}

func newFunctionType(pointer *C.wasm_functype_t, ownedBy interface{}) *FunctionType {
	functionType := &FunctionType{_inner: pointer, _ownedBy: ownedBy}

	if ownedBy == nil {
		runtime.SetFinalizer(functionType, func(functionType *FunctionType) {
			C.wasm_functype_delete(functionType.inner())
		})
	}

	return functionType
}

// NewFunctionType instantiates a new FunctionType from two ValueType arrays: the parameters and the results.
//
//   params := wasmer.NewValueTypes()
//   results := wasmer.NewValueTypes(wasmer.I32)
//   functionType := wasmer.NewFunctionType(params, results)
//
func NewFunctionType(params []*ValueType, results []*ValueType) *FunctionType {
	paramsAsValueTypeVec := toValueTypeVec(params)
	resultsAsValueTypeVec := toValueTypeVec(results)

	pointer := C.wasm_functype_new(&paramsAsValueTypeVec, &resultsAsValueTypeVec)

	return newFunctionType(pointer, nil)
}

func (self *FunctionType) inner() *C.wasm_functype_t {
	return self._inner
}

func (self *FunctionType) ownedBy() interface{} {
	if self._ownedBy == nil {
		return self
	}

	return self._ownedBy
}

// Params returns the parameters definitions from the FunctionType as a ValueType array
//
//   params := wasmer.NewValueTypes()
//   results := wasmer.NewValueTypes(wasmer.I32)
//   functionType := wasmer.NewFunctionType(params, results)
//   paramsValueTypes = functionType.Params()
//
func (self *FunctionType) Params() []*ValueType {
	return toValueTypeList(C.wasm_functype_params(self.inner()), self.ownedBy())
}

// Results returns the results definitions from the FunctionType as a ValueType array
//
//   params := wasmer.NewValueTypes()
//   results := wasmer.NewValueTypes(wasmer.I32)
//   functionType := wasmer.NewFunctionType(params, results)
//   resultsValueTypes = functionType.Results()
//
func (self *FunctionType) Results() []*ValueType {
	return toValueTypeList(C.wasm_functype_results(self.inner()), self.ownedBy())
}

// IntoExternType converts the FunctionType into an ExternType.
//
//   function, _ := instance.Exports.GetFunction("exported_function")
//   functionType := function.Type()
//   externType = functionType.IntoExternType()
//
func (self *FunctionType) IntoExternType() *ExternType {
	pointer := C.wasm_functype_as_externtype_const(self.inner())

	return newExternType(pointer, self.ownedBy())
}
