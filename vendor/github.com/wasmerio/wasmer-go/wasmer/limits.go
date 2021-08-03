package wasmer

// #include <wasmer_wasm.h>
//
// uint32_t to_limit_max_unbound() {
//     return wasm_limits_max_default;
// }
import "C"
import "runtime"

// LimitMaxUnbound returns the value used to represent an unbound
// limit, i.e. when a limit only has a min but not a max. See Limit.
func LimitMaxUnbound() uint32 {
	return uint32(C.to_limit_max_unbound())
}

// Limits classify the size range of resizeable storage associated
// with memory types and table types.
//
// See also
//
// Specification: https://webassembly.github.io/spec/core/syntax/types.html#limits
type Limits struct {
	_inner C.wasm_limits_t
}

func newLimits(pointer *C.wasm_limits_t, ownedBy interface{}) *Limits {
	limits, err := NewLimits(uint32(pointer.min), uint32(pointer.max))

	if err != nil {
		return nil
	}

	if ownedBy != nil {
		runtime.KeepAlive(ownedBy)
	}

	return limits
}

// NewLimits instantiates a new Limits which describes the Memory used.
// The minimum and maximum parameters are "number of memory pages".
//
// ️Note: Each page is 64 KiB in size.
//
// Note: You cannot Memory.Grow the Memory beyond the maximum defined here.
func NewLimits(minimum uint32, maximum uint32) (*Limits, error) {
	if minimum > maximum {
		return nil, newErrorWith("The minimum limit is greater than the maximum one")
	}

	return &Limits{
		_inner: C.wasm_limits_t{
			min: C.uint32_t(minimum),
			max: C.uint32_t(maximum),
		},
	}, nil
}

func (self *Limits) inner() *C.wasm_limits_t {
	return &self._inner
}

// Minimum returns the minimum size of the Memory allocated in "number of pages".
//
// Note:️ Each page is 64 KiB in size.
func (self *Limits) Minimum() uint32 {
	return uint32(self.inner().min)
}

// Maximum returns the maximum size of the Memory allocated in "number of pages".
//
// Each page is 64 KiB in size.
//
// Note: You cannot Memory.Grow beyond this defined maximum size.
func (self *Limits) Maximum() uint32 {
	return uint32(self.inner().max)
}
