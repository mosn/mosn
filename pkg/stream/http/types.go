package http

import "github.com/valyala/fasthttp"

type requestHeader struct {
	*fasthttp.RequestHeader
}

// Get value of key
func (h requestHeader) Get(key string) (string, bool) {
	result := h.Peek(key)
	if result != nil {
		return string(result), true
	}
	return "", false
}

// Set key-value pair in header map, the previous pair will be replaced if exists
func (h requestHeader) Set(key string, value string) {
	h.RequestHeader.Set(key, value)
}

// Del delete pair of specified key
func (h requestHeader) Del(key string) {
	h.RequestHeader.Del(key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (h requestHeader) Range(f func(key, value string) bool) {
	stopped := false
	h.VisitAll(func(key, value []byte) {
		if stopped {
			return
		}
		stopped = !f(string(key), string(value))
	})
}

func (h requestHeader) ByteSize() (size uint64) {
	h.VisitAll(func(key, value []byte) {
		size += uint64(len(key) + len(value))
	})
	return size
}

type responseHeader struct {
	*fasthttp.ResponseHeader
}

// Get value of key
func (h responseHeader) Get(key string) (value string, ok bool) {
	result := h.Peek(key)
	if result != nil {
		value = string(result)
		return
	}
	return
}

// Set key-value pair in header map, the previous pair will be replaced if exists
func (h responseHeader) Set(key string, value string) {
	h.ResponseHeader.Set(key, value)
}

// Del delete pair of specified key
func (h responseHeader) Del(key string) {
	h.ResponseHeader.Del(key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (h responseHeader) Range(f func(key, value string) bool) {
	stopped := false
	h.VisitAll(func(key, value []byte) {
		if stopped {
			return
		}
		stopped = !f(string(key), string(value))
	})
}

func (h responseHeader) ByteSize() (size uint64) {
	h.VisitAll(func(key, value []byte) {
		size += uint64(len(key) + len(value))
	})
	return size
}
