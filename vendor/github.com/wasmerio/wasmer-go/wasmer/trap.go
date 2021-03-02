package wasmer

// #include <wasmer_wasm.h>
//
// #define own
//
// own wasm_trap_t* to_wasm_trap_new(wasm_store_t *store, uint8_t *message_bytes, size_t message_length) {
//     // `wasm_message_t` is an alias to `wasm_byte_vec_t`.
//     wasm_message_t message;
//     message.size = message_length;
//     message.data = (wasm_byte_t*) message_bytes;
//
//     return wasm_trap_new(store, &message);
// }
import "C"
import (
	"runtime"
	"unsafe"
)

// Trap stores trace message with backtrace when an error happened.
type Trap struct {
	_inner   *C.wasm_trap_t
	_ownedBy interface{}
}

func newTrap(pointer *C.wasm_trap_t, ownedBy interface{}) *Trap {
	trap := &Trap{
		_inner:   pointer,
		_ownedBy: ownedBy,
	}

	if ownedBy == nil {
		runtime.SetFinalizer(trap, func(trap *Trap) {
			inner := trap.inner()

			if inner != nil {
				C.wasm_trap_delete(inner)
			}
		})
	}

	return trap
}

// Creates a new trap with a message.
//
//   engine := wasmer.NewEngine()
//   store := wasmer.NewStore(engine)
//   trap := NewTrap(store, "oops")
func NewTrap(store *Store, message string) *Trap {
	messageBytes := []byte(message)
	var bytesPointer *C.uint8_t
	bytesLength := len(messageBytes)

	if bytesLength > 0 {
		bytesPointer = (*C.uint8_t)(unsafe.Pointer(&messageBytes[0]))
	}

	trap := C.to_wasm_trap_new(store.inner(), bytesPointer, C.size_t(bytesLength))

	runtime.KeepAlive(store)
	runtime.KeepAlive(message)

	return newTrap(trap, nil)
}

func (self *Trap) inner() *C.wasm_trap_t {
	return self._inner
}

func (self *Trap) ownedBy() interface{} {
	if self._ownedBy == nil {
		return self
	}

	return self._ownedBy
}

// Message returns the message attached to the current Trap.
func (self *Trap) Message() string {
	var bytes C.wasm_byte_vec_t
	C.wasm_trap_message(self.inner(), &bytes)

	runtime.KeepAlive(self)

	goBytes := C.GoBytes(unsafe.Pointer(bytes.data), C.int(bytes.size)-1)
	C.wasm_byte_vec_delete(&bytes)

	return string(goBytes)
}

// Origin returns the top frame of WebAssembly stack responsible for
// this trap.
//
//     frame := trap.Origin()
func (self *Trap) Origin() *Frame {
	frame := C.wasm_trap_origin(self.inner())

	runtime.KeepAlive(self)

	if frame == nil {
		return nil
	}

	return newFrame(frame, self.ownedBy())
}

// Trace returns the trace of WebAssembly frames for this trap.
func (self *Trap) Trace() *Trace {
	return newTrace(self)
}

// Frame represents a frame of a WebAssembly stack trace.
type Frame struct {
	_inner   *C.wasm_frame_t
	_ownedBy interface{}
}

func newFrame(pointer *C.wasm_frame_t, ownedBy interface{}) *Frame {
	frame := &Frame{
		_inner:   pointer,
		_ownedBy: ownedBy,
	}

	if ownedBy == nil {
		runtime.SetFinalizer(frame, func(frame *Frame) {
			C.wasm_frame_delete(frame.inner())
		})
	}

	return frame
}

func (self *Frame) inner() *C.wasm_frame_t {
	return self._inner
}

func (self *Frame) ownedBy() interface{} {
	if self._ownedBy == nil {
		return self
	}

	return self._ownedBy
}

// FunctionIndex returns the function index in the original
// WebAssembly module that this frame corresponds to.
func (self *Frame) FunctionIndex() uint32 {
	index := C.wasm_frame_func_index(self.inner())

	runtime.KeepAlive(self)

	return uint32(index)
}

// FunctionOffset returns the byte offset from the beginning of the
// function in the original WebAssembly file to the instruction this
// frame points to.
func (self *Frame) FunctionOffset() uint {
	index := C.wasm_frame_func_offset(self.inner())

	runtime.KeepAlive(self)

	return uint(index)
}

func (self *Frame) Instance() {
	//TODO: See https://github.com/wasmerio/wasmer/blob/6fbc903ea32774c830fd9ee86140d1406ac5d745/lib/c-api/src/wasm_c_api/types/frame.rs#L31-L34
	panic("to do!")
}

// ModuleOffset returns the byte offset from the beginning of the
// original WebAssembly file to the instruction this frame points to.
func (self *Frame) ModuleOffset() uint {
	index := C.wasm_frame_module_offset(self.inner())

	runtime.KeepAlive(self)

	return uint(index)
}

// Trace represents a WebAssembly trap.
type Trace struct {
	_inner C.wasm_frame_vec_t
	frames []*Frame
}

func newTrace(trap *Trap) *Trace {
	var self = &Trace{}
	C.wasm_trap_trace(trap.inner(), self.inner())

	runtime.KeepAlive(trap)
	runtime.SetFinalizer(self, func(self *Trace) {
		C.wasm_frame_vec_delete(self.inner())
	})

	numberOfFrames := int(self.inner().size)
	frames := make([]*Frame, numberOfFrames)
	firstFrame := unsafe.Pointer(self.inner().data)
	sizeOfFramePointer := unsafe.Sizeof(firstFrame)

	var currentFramePointer **C.wasm_frame_t

	for nth := 0; nth < numberOfFrames; nth++ {
		currentFramePointer = (**C.wasm_frame_t)(unsafe.Pointer(uintptr(firstFrame) + uintptr(nth)*sizeOfFramePointer))
		frames[nth] = newFrame(*currentFramePointer, self)
	}

	self.frames = frames

	return self
}

func (self *Trace) inner() *C.wasm_frame_vec_t {
	return &self._inner
}
