package wasmer

// #include <wasmer_wasm.h>
import "C"
import "runtime"

type Target struct {
	_inner *C.wasm_target_t
}

func newTarget(target *C.wasm_target_t) *Target {
	self := &Target{
		_inner: target,
	}

	runtime.SetFinalizer(self, func(self *Target) {
		C.wasm_target_delete(self.inner())
	})

	return self
}

func NewTarget(triple *Triple, cpuFeatures *CpuFeatures) *Target {
	return newTarget(C.wasm_target_new(triple.inner(), cpuFeatures.inner()))
}

func (self *Target) inner() *C.wasm_target_t {
	return self._inner
}

type Triple struct {
	_inner *C.wasm_triple_t
}

func newTriple(triple *C.wasm_triple_t) *Triple {
	self := &Triple{
		_inner: triple,
	}

	runtime.SetFinalizer(self, func(self *Triple) {
		C.wasm_triple_delete(self.inner())
	})

	return self
}

func NewTriple(triple string) (*Triple, error) {
	cTripleName := newName(triple)
	defer C.wasm_name_delete(&cTripleName)

	var cTriple *C.wasm_triple_t

	err := maybeNewErrorFromWasmer(func() bool {
		cTriple := C.wasm_triple_new(&cTripleName)

		return cTriple == nil
	})

	if err != nil {
		return nil, err
	}

	return newTriple(cTriple), nil
}

func NewTripleFromHost() *Triple {
	return newTriple(C.wasm_triple_new_from_host())
}

func (self *Triple) inner() *C.wasm_triple_t {
	return self._inner
}

type CpuFeatures struct {
	_inner *C.wasm_cpu_features_t
}

func newCpuFeatures(cpu_features *C.wasm_cpu_features_t) *CpuFeatures {
	self := &CpuFeatures{
		_inner: cpu_features,
	}

	runtime.SetFinalizer(self, func(self *CpuFeatures) {
		C.wasm_cpu_features_delete(self.inner())
	})

	return self
}

func NewCpuFeatures() *CpuFeatures {
	return newCpuFeatures(C.wasm_cpu_features_new())
}

func (self *CpuFeatures) Add(feature string) error {
	cFeature := newName(feature)
	defer C.wasm_name_delete(&cFeature)

	err := maybeNewErrorFromWasmer(func() bool {
		return false == C.wasm_cpu_features_add(self.inner(), &cFeature)
	})

	if err != nil {
		return err
	}

	return nil
}

func (self *CpuFeatures) inner() *C.wasm_cpu_features_t {
	return self._inner
}
