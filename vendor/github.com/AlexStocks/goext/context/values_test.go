package gxcontext

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestNewValuesContext(t *testing.T) {
	xassert := assert.New(t)

	ctx := NewValuesContext(nil)
	key := "hello key"
	value := "hello value"
	ctx.Set(key, value)
	v, b := ctx.Get(key)
	xassert.Equalf(value, v, "key:%s", key)
	xassert.Equalf(true, b, "key:%s", key)

	ctx.Delete(key)
	v, b = ctx.Get(key)
	xassert.Equalf(nil, v, "key:%s", key)
	xassert.Equalf(false, b, "key:%s", key)

	ctx1 := NewValuesContext(nil)
	key1 := struct {
		s string
		i int
	}{s: "hello key", i: 100}
	ctx1.Set(key1, value)
	v1, b1 := ctx1.Get(key1)
	xassert.Equalf(value, v1, "key:%#v", key1)
	xassert.Equalf(true, b1, "key:%#v", key1)

	ctx1.Delete(key1)
	v1, b1 = ctx1.Get(key)
	xassert.Equalf(nil, v1, "key:%#v", key1)
	xassert.Equalf(false, b1, "key:%#v", key1)
}
