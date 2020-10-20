package xprotocol

import "sync"

// a map with sync.Map API
type syncStreamMap struct {
	m map[uint64]*xStream
	mux sync.RWMutex
}

func NewStreamMap(size int) *syncStreamMap {
	return &syncStreamMap{
		m : make(map[uint64]*xStream, size),
	}
}

func (sm *syncStreamMap) Range(f func(key uint64, value *xStream)) {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	for k, v := range sm.m {
		f(k, v)
	}
}

func (sm *syncStreamMap) Load(key uint64) (*xStream, bool){
	sm.mux.RLock()
	defer sm.mux.RUnlock()
	str, ok := sm.m[key]
	return str, ok
}

func (sm *syncStreamMap) Store(key uint64, val *xStream) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.m[key] = val
}

func (sm *syncStreamMap) Delete(key uint64) {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	delete(sm.m, key)
}

func (sm *syncStreamMap) LoadAndDelete(key uint64) (value *xStream, loaded bool){
	sm.mux.Lock()
	defer sm.mux.Unlock()

	str, ok := sm.m[key]
	if ok {
		delete(sm.m, key)
	}

	return str, ok
}

func (sm *syncStreamMap) Len() int {
	sm.mux.RLock()
	defer sm.mux.RUnlock()
	return len(sm.m)
}
