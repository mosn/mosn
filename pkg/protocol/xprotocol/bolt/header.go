package bolt

import (
	"reflect"
	"unsafe"
)

type bytesKV struct {
	key   []byte
	value []byte
}

type header struct {
	kvs []bytesKV
}

// ~ HeaderMap
func (h *header) Get(key string) (value string, ok bool) {
	for i, n := 0, len(h.kvs); i < n; i++ {
		kv := &h.kvs[i]
		if key == string(kv.key) {
			return string(kv.value), true
		}
	}
	return "", false
}

func (h *header) Set(key string, value string) {
	for i, n := 0, len(h.kvs); i < n; i++ {
		kv := &h.kvs[i]
		if key == string(kv.key) {
			kv.value = append(kv.value[:0], value...)
			return
		}
	}

	var kv *bytesKV
	h.kvs, kv = allocKV(h.kvs)
	kv.key = append(kv.key[:0], key...)
	kv.value = append(kv.value[:0], value...)
}

func (h *header) Add(key string, value string) {
	panic("not supported")
}

func (h *header) Del(key string) {
	for i, n := 0, len(h.kvs); i < n; i++ {
		kv := &h.kvs[i]
		if key == string(kv.key) {
			tmp := *kv
			copy(h.kvs[i:], h.kvs[i+1:])
			n--
			h.kvs[n] = tmp
			h.kvs = h.kvs[:n]
			return
		}
	}
}

func (h *header) Range(f func(key, value string) bool) {
	for i, n := 0, len(h.kvs); i < n; i++ {
		kv := &h.kvs[i]
		// false means stop iteration
		if !f(b2s(kv.key), b2s(kv.value)) {
			return
		}
	}
}

func (h *header) Clone() *header {
	n := len(h.kvs)

	clone := &header{
		kvs: make([]bytesKV, n),
	}

	for i := 0; i < n; i++ {
		src := &h.kvs[i]
		dst := &clone.kvs[i]

		dst.key = append(dst.key[:0], src.key...)
		dst.value = append(dst.value[:0], src.value...)
	}

	return clone
}

func (h *header) ByteSize() (size uint64) {
	for _, kv := range h.kvs {
		size += uint64(len(kv.key) + len(kv.value))
	}
	return
}

func allocKV(h []bytesKV) ([]bytesKV, *bytesKV) {
	n := len(h)
	if cap(h) > n {
		h = h[:n+1]
	} else {
		h = append(h, bytesKV{})
	}
	return h, &h[n]
}

// b2s converts byte slice to a string without memory allocation.
// See https://groups.google.com/forum/#!msg/Golang-Nuts/ENgbUzYvCuU/90yGx7GUAgAJ .
//
// Note it may break if string and/or slice header will change
// in the future go versions.
func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// s2b converts string to a byte slice without memory allocation.
//
// Note it may break if string and/or slice header will change
// in the future go versions.
func s2b(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	return *(*[]byte)(unsafe.Pointer(&bh))
}
