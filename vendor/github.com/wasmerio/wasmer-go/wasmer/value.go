package wasmer

// #include <wasmer_wasm.h>
//
// int32_t to_int32(wasm_val_t *value) {
//     return value->of.i32;
// }
//
// int64_t to_int64(wasm_val_t *value) {
//     return value->of.i64;
// }
//
// float32_t to_float32(wasm_val_t *value) {
//     return value->of.f32;
// }
//
// float64_t to_float64(wasm_val_t *value) {
//     return value->of.f64;
// }
//
// wasm_ref_t *to_ref(wasm_val_t *value) {
//     return value->of.ref;
// }
import "C"
import (
	"fmt"
	"unsafe"
)

type Value struct {
	_inner *C.wasm_val_t
}

func newValue(pointer *C.wasm_val_t) Value {
	return Value{_inner: pointer}
}

// NewValue instantiates a new Value with the given value and ValueKind.
//
// ⚠️ If a Wasm value cannot be created from the given value, NewValue will panic.
//
//   value := NewValue(42, I32)
//
func NewValue(value interface{}, kind ValueKind) Value {
	output, err := fromGoValue(value, kind)

	if err != nil {
		panic(fmt.Sprintf("Cannot create a Wasm `%s` value from `%T`", err, value))
	}

	return newValue(&output)
}

// NewI32 instantiates a new I32 Value with the given value.
//
// ⚠️ If a Wasm value cannot be created from the given value, NewI32 will panic.
//
//   value := NewI32(42)
//
func NewI32(value interface{}) Value {
	return NewValue(value, I32)
}

// NewI64 instantiates a new I64 Value with the given value.
//
// ⚠️ If a Wasm value cannot be created from the given value, NewI64 will panic.
//
//   value := NewI64(42)
//
func NewI64(value interface{}) Value {
	return NewValue(value, I64)
}

// NewF32 instantiates a new F32 Value with the given value.
//
// ⚠️ If a Wasm value cannot be created from the given value, NewF32 will panic.
//
//   value := NewF32(4.2)
//
func NewF32(value interface{}) Value {
	return NewValue(value, F32)
}

// NewF64 instantiates a new F64 Value with the given value.
//
// ⚠️ If a Wasm value cannot be created from the given value, NewF64 will panic.
//
//   value := NewF64(4.2)
//
func NewF64(value interface{}) Value {
	return NewValue(value, F64)
}

func (self *Value) inner() *C.wasm_val_t {
	return self._inner
}

// Kind returns the Value's ValueKind.
//
//   value := NewF64(4.2)
//   _ = value.Kind()
//
func (self *Value) Kind() ValueKind {
	return ValueKind(self.inner().kind)
}

// Unwrap returns the Value's value as a native Go value.
//
//   value := NewF64(4.2)
//   _ = value.Unwrap()
//
func (self *Value) Unwrap() interface{} {
	return toGoValue(self.inner())
}

// I32 returns the Value's value as a native Go int32.
//
//   value := NewI32(42)
//   _ = value.I32()
//
func (self *Value) I32() int32 {
	pointer := self.inner()

	if ValueKind(pointer.kind) != I32 {
		panic("Cannot convert value to `int32`")
	}

	return int32(C.to_int32(pointer))
}

// I64 returns the Value's value as a native Go int64.
//
//   value := NewI64(42)
//   _ = value.I64()
//
func (self *Value) I64() int64 {
	pointer := self.inner()

	if ValueKind(pointer.kind) != I64 {
		panic("Cannot convert value to `int64`")
	}

	return int64(C.to_int64(pointer))
}

// F32 returns the Value's value as a native Go float32.
//
//   value := NewF32(4.2)
//   _ = value.F32()
//
func (self *Value) F32() float32 {
	pointer := self.inner()

	if ValueKind(pointer.kind) != F32 {
		panic("Cannot convert value to `float32`")
	}

	return float32(C.to_float32(pointer))
}

// F64 returns the Value's value as a native Go float64.
//
//   value := NewF64(4.2)
//   _ = value.F64()
//
func (self *Value) F64() float64 {
	pointer := self.inner()

	if ValueKind(pointer.kind) != F64 {
		panic("Cannot convert value to `float64`")
	}

	return float64(C.to_float64(pointer))
}

func toGoValue(pointer *C.wasm_val_t) interface{} {
	switch ValueKind(pointer.kind) {
	case I32:
		return int32(C.to_int32(pointer))
	case I64:
		return int64(C.to_int64(pointer))
	case F32:
		return float32(C.to_float32(pointer))
	case F64:
		return float64(C.to_float64(pointer))
	default:
		panic("to do `newValue`")
	}
}

func fromGoValue(value interface{}, kind ValueKind) (C.wasm_val_t, error) {
	output := C.wasm_val_t{}

	switch kind {
	case I32:
		output.kind = kind.inner()

		var of = (*int32)(unsafe.Pointer(&output.of))

		switch value.(type) {
		case int8:
			*of = int32(value.(int8))
		case uint8:
			*of = int32(value.(uint8))
		case int16:
			*of = int32(value.(int16))
		case uint16:
			*of = int32(value.(uint16))
		case int32:
			*of = value.(int32)
		case int:
			*of = int32(value.(int))
		case uint:
			*of = int32(value.(uint))
		default:
			return output, newErrorWith("i32")
		}
	case I64:
		output.kind = kind.inner()

		var of = (*int64)(unsafe.Pointer(&output.of))

		switch value.(type) {
		case int8:
			*of = int64(value.(int8))
		case uint8:
			*of = int64(value.(uint8))
		case int16:
			*of = int64(value.(int16))
		case uint16:
			*of = int64(value.(uint16))
		case int32:
			*of = int64(value.(int32))
		case uint32:
			*of = int64(value.(int64))
		case int64:
			*of = value.(int64)
		case int:
			*of = int64(value.(int))
		case uint:
			*of = int64(value.(uint))
		default:
			return output, newErrorWith("i64")
		}
	case F32:
		output.kind = kind.inner()

		var of = (*float32)(unsafe.Pointer(&output.of))

		switch value.(type) {
		case float32:
			*of = value.(float32)
		default:
			return output, newErrorWith("f32")
		}
	case F64:
		output.kind = kind.inner()

		var of = (*float64)(unsafe.Pointer(&output.of))

		switch value.(type) {
		case float32:
			*of = float64(value.(float32))
		case float64:
			*of = value.(float64)
		default:
			return output, newErrorWith("f64")
		}
	default:
		panic("To do, `fromGoValue`!")
	}

	return output, nil
}

func toValueVec(list []Value, vec *C.wasm_val_vec_t) {
	numberOfValues := len(list)
	values := make([]C.wasm_val_t, numberOfValues)

	for nth, item := range list {
		value, err := fromGoValue(item.I32(), item.Kind())

		if err != nil {
			panic(err)
		}

		values[nth] = value
	}

	if numberOfValues > 0 {
		C.wasm_val_vec_new(vec, C.size_t(numberOfValues), (*C.wasm_val_t)(unsafe.Pointer(&values[0])))
	}
}

func toValueList(values *C.wasm_val_vec_t) []Value {
	numberOfValues := int(values.size)
	list := make([]Value, numberOfValues)
	firstValue := unsafe.Pointer(values.data)
	sizeOfValuePointer := unsafe.Sizeof(C.wasm_val_t{})

	var currentValuePointer *C.wasm_val_t

	for nth := 0; nth < numberOfValues; nth++ {
		currentValuePointer = (*C.wasm_val_t)(unsafe.Pointer(uintptr(firstValue) + uintptr(nth)*sizeOfValuePointer))
		value := newValue(currentValuePointer)
		list[nth] = value
	}

	return list
}
