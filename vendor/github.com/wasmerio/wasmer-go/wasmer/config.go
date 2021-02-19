package wasmer

// #include <wasmer_wasm.h>
import "C"

type Config struct {
	_inner *C.wasm_config_t
}

// NewConfig instantiates and returns a new Config.
//
//   config := NewConfig()
//
func NewConfig() *Config {
	config := C.wasm_config_new()

	return &Config{
		_inner: config,
	}
}

func (self *Config) inner() *C.wasm_config_t {
	return self._inner
}

// UseJITEngine sets the engine to JIT in the configuration.
//
//   config := NewConfig()
//   config.UseJITEngine()
//
func (self *Config) UseJITEngine() {
	C.wasm_config_set_engine(self.inner(), C.JIT)
}

// UseNativeEngine sets the engine to Native in the configuration.
//
//   config := NewConfig()
//   config.UseNativeEngine()
//
func (self *Config) UseNativeEngine() {
	C.wasm_config_set_engine(self.inner(), C.NATIVE)
}

// UseCraneliftCompiler sets the compiler to Cranelift in the configuration.
//
//   config := NewConfig()
//   config.UseCraneliftCompiler()
//
func (self *Config) UseCraneliftCompiler() {
	C.wasm_config_set_engine(self.inner(), C.CRANELIFT)
}

// UseLLVMCompiler sets the compiler to LLVM in the configuration.
//
//   config := NewConfig()
//   config.UseLLVMCompiler()
//
func (self *Config) UseLLVMCompiler() {
	C.wasm_config_set_engine(self.inner(), C.LLVM)
}

func (self *Config) UseTarget(target *Target) {
	C.wasm_config_set_target(self.inner(), target.inner())
}
