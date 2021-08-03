package wasmer

// #include <wasmer_wasm.h>
import "C"
import (
	"runtime"
	"unsafe"
)

// ValueKind represents the kind of a value.
type ValueKind C.wasm_valkind_t

const (
	// A 32-bit integer. In WebAssembly, integers are
	// sign-agnostic, i.E. this can either be signed or unsigned.
	I32 = ValueKind(C.WASM_I32)

	// A 64-bit integer. In WebAssembly, integers are
	// sign-agnostic, i.E. this can either be signed or unsigned.
	I64 = ValueKind(C.WASM_I64)

	// A 32-bit float.
	F32 = ValueKind(C.WASM_F32)

	// A 64-bit float.
	F64 = ValueKind(C.WASM_F64)

	// An externref value which can hold opaque data to the
	// WebAssembly instance itself.
	AnyRef = ValueKind(C.WASM_ANYREF)

	// A first-class reference to a WebAssembly function.
	FuncRef = ValueKind(C.WASM_FUNCREF)
)

// String returns the ValueKind as a string.
//
//   I32.String()     // "i32"
//   I64.String()     // "i64"
//   F32.String()     // "f32"
//   F64.String()     // "f64"
//   AnyRef.String()  // "anyref"
//   FuncRef.String() // "funcref"
func (self ValueKind) String() string {
	switch self {
	case I32:
		return "i32"
	case I64:
		return "i64"
	case F32:
		return "f32"
	case F64:
		return "f64"
	case AnyRef:
		return "anyref"
	case FuncRef:
		return "funcref"
	}
	panic("Unknown value kind")
}

// IsNumber returns true if the ValueKind is a number type.
//
//   I32.IsNumber()     // true
//   I64.IsNumber()     // true
//   F32.IsNumber()     // true
//   F64.IsNumber()     // true
//   AnyRef.IsNumber()  // false
//   FuncRef.IsNumber() // false
func (self ValueKind) IsNumber() bool {
	return bool(C.wasm_valkind_is_num(C.wasm_valkind_t(self)))
}

// IsReference returns true if the ValueKind is a reference.
//
//   I32.IsReference()     // false
//   I64.IsReference()     // false
//   F32.IsReference()     // false
//   F64.IsReference()     // false
//   AnyRef.IsReference()  // true
//   FuncRef.IsReference() // true
func (self ValueKind) IsReference() bool {
	return bool(C.wasm_valkind_is_ref(C.wasm_valkind_t(self)))
}

func (self ValueKind) inner() C.wasm_valkind_t {
	return C.wasm_valkind_t(self)
}

// ValueType classifies the individual values that WebAssembly code
// can compute with and the values that a variable accepts.
type ValueType struct {
	_inner   *C.wasm_valtype_t
	_ownedBy interface{}
}

// NewValueType instantiates a new ValueType given a ValueKind.
//
//   valueType := NewValueType(I32)
func NewValueType(kind ValueKind) *ValueType {
	pointer := C.wasm_valtype_new(C.wasm_valkind_t(kind))

	return newValueType(pointer, nil)
}

func newValueType(pointer *C.wasm_valtype_t, ownedBy interface{}) *ValueType {
	valueType := &ValueType{_inner: pointer, _ownedBy: ownedBy}

	if ownedBy == nil {
		runtime.SetFinalizer(valueType, func(valueType *ValueType) {
			C.wasm_valtype_delete(valueType.inner())
		})
	}

	return valueType
}

func (self *ValueType) inner() *C.wasm_valtype_t {
	return self._inner
}

// Kind returns the ValueType's ValueKind
//
//   valueType := NewValueType(I32)
//   _ = valueType.Kind()
func (self *ValueType) Kind() ValueKind {
	kind := ValueKind(C.wasm_valtype_kind(self.inner()))

	runtime.KeepAlive(self)

	return kind
}

// NewValueTypes instantiates a new ValueType array from a list of
// ValueKind. Not that the list may be empty.
//
//   valueTypes := NewValueTypes(I32, I64, F32)
//
// Note:️ NewValueTypes is specifically designed to help you declare
// function types, e.g. with NewFunctionType:
//
//   functionType := NewFunctionType(
//   	NewValueTypes(), // arguments
//   	NewValueTypes(I32), // results
//   )
func NewValueTypes(kinds ...ValueKind) []*ValueType {
	valueTypes := make([]*ValueType, len(kinds))

	for nth, kind := range kinds {
		valueTypes[nth] = NewValueType(kind)
	}

	return valueTypes
}

func toValueTypeVec(valueTypes []*ValueType) C.wasm_valtype_vec_t {
	vec := C.wasm_valtype_vec_t{}
	C.wasm_valtype_vec_new_uninitialized(&vec, C.size_t(len(valueTypes)))

	firstValueTypePointer := unsafe.Pointer(vec.data)

	for nth, valueType := range valueTypes {
		pointer := C.wasm_valtype_new(C.wasm_valtype_kind(valueType.inner()))
		*(**C.wasm_valtype_t)(unsafe.Pointer(uintptr(firstValueTypePointer) + unsafe.Sizeof(pointer)*uintptr(nth))) = pointer
	}

	runtime.KeepAlive(valueTypes)

	return vec
}

func toValueTypeList(valueTypes *C.wasm_valtype_vec_t, ownedBy interface{}) []*ValueType {
	numberOfValueTypes := int(valueTypes.size)
	list := make([]*ValueType, numberOfValueTypes)
	firstValueType := unsafe.Pointer(valueTypes.data)
	sizeOfValueTypePointer := unsafe.Sizeof(firstValueType)

	var currentValueTypePointer *C.wasm_valtype_t

	for nth := 0; nth < numberOfValueTypes; nth++ {
		currentValueTypePointer = *(**C.wasm_valtype_t)(unsafe.Pointer(uintptr(firstValueType) + uintptr(nth)*sizeOfValueTypePointer))
		valueType := newValueType(currentValueTypePointer, ownedBy)
		list[nth] = valueType
	}

	return list
}
