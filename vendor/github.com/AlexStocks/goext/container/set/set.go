// Copyright (C) 2017 ScyllaDB
// Use of this source code is governed by a ALv2-style
// license that can be found in the LICENSE file.

package gxset

import (
	"github.com/scylladb/go-set/b16set"
	"github.com/scylladb/go-set/b32set"
	"github.com/scylladb/go-set/b64set"
	"github.com/scylladb/go-set/b8set"
	"github.com/scylladb/go-set/f32set"
	"github.com/scylladb/go-set/f64set"
	"github.com/scylladb/go-set/i16set"
	"github.com/scylladb/go-set/i32set"
	"github.com/scylladb/go-set/i64set"
	"github.com/scylladb/go-set/i8set"
	"github.com/scylladb/go-set/iset"
	"github.com/scylladb/go-set/strset"
	"github.com/scylladb/go-set/u16set"
	"github.com/scylladb/go-set/u32set"
	"github.com/scylladb/go-set/u64set"
	"github.com/scylladb/go-set/u8set"
	"github.com/scylladb/go-set/uset"
)

//go:generate mkdir -p b8set
//go:generate go_generics -i internal/set/set.go -t T=[8]byte -o b8set/b8set.go -p b8set
//go:generate go_generics -i internal/set/set_test.go -t P=[8]byte -o b8set/b8set_test.go -p b8set

// NewByte8Set is a convenience function to create a new b16set.Set
func NewByte8Set() *b8set.Set {
	return b8set.New()
}

//go:generate mkdir -p b16set
//go:generate go_generics -i internal/set/set.go -t T=[16]byte -o b16set/b16set.go -p b16set
//go:generate go_generics -i internal/set/set_test.go -t P=[16]byte -o b16set/b16set_test.go -p b16set

// NewByte16Set is a convenience function to create a new b16set.Set
func NewByte16Set() *b16set.Set {
	return b16set.New()
}

//go:generate mkdir -p b32set
//go:generate go_generics -i internal/set/set.go -t T=[32]byte -o b32set/b32set.go -p b32set
//go:generate go_generics -i internal/set/set_test.go -t P=[32]byte -o b32set/b32set_test.go -p b32set

// NewByte32Set is a convenience function to create a new b32set.Set
func NewByte32Set() *b32set.Set {
	return b32set.New()
}

//go:generate mkdir -p b64set
//go:generate go_generics -i internal/set/set.go -t T=[64]byte -o b64set/b64set.go -p b64set
//go:generate go_generics -i internal/set/set_test.go -t P=[64]byte -o b64set/b64set_test.go -p b64set

// NewByte64Set is a convenience function to create a new b64set.Set
func NewByte64Set() *b64set.Set {
	return b64set.New()
}

//go:generate mkdir -p f32set
//go:generate go_generics -i internal/set/set.go -t T=float32 -o f32set/f32set.go -p f32set
//go:generate go_generics -i internal/set/set_test.go -t P=float32 -o f32set/f32set_test.go -p f32set

// NewFloat32Set is a convenience function to create a new f32set.Set
func NewFloat32Set() *f32set.Set {
	return f32set.New()
}

//go:generate mkdir -p f64set
//go:generate go_generics -i internal/set/set.go -t T=float64 -o f64set/f64set.go -p f64set
//go:generate go_generics -i internal/set/set_test.go -t P=float64 -o f64set/f64set_test.go -p f64set

// NewFloat64Set is a convenience function to create a new f64set.Set
func NewFloat64Set() *f64set.Set {
	return f64set.New()
}

//go:generate mkdir -p iset
//go:generate go_generics -i internal/set/set.go -t T=int -o iset/iset.go -p iset
//go:generate go_generics -i internal/set/set_test.go -t P=int -o iset/iset_test.go -p iset

// NewIntSet is a convenience function to create a new iset.Set
func NewIntSet() *iset.Set {
	return iset.New()
}

//go:generate mkdir -p i8set
//go:generate go_generics -i internal/set/set.go -t T=int8 -o i8set/i8set.go -p i8set
//go:generate go_generics -i internal/set/set_test.go -t P=int8 -o i8set/i8set_test.go -p i8set

// NewInt8Set is a convenience function to create a new i8set.Set
func NewInt8Set() *i8set.Set {
	return i8set.New()
}

//go:generate mkdir -p i16set
//go:generate go_generics -i internal/set/set.go -t T=int16 -o i16set/i16set.go -p i16set
//go:generate go_generics -i internal/set/set_test.go -t P=int16 -o i16set/i16set_test.go -p i16set

// NewInt16Set is a convenience function to create a new i16set.Set
func NewInt16Set() *i16set.Set {
	return i16set.New()
}

//go:generate mkdir -p i32set
//go:generate go_generics -i internal/set/set.go -t T=int32 -o i32set/i32set.go -p i32set
//go:generate go_generics -i internal/set/set_test.go -t P=int32 -o i32set/i32set_test.go -p i32set

// NewInt32Set is a convenience function to create a new i32set.Set
func NewInt32Set() *i32set.Set {
	return i32set.New()
}

//go:generate mkdir -p i64set
//go:generate go_generics -i internal/set/set.go -t T=int64 -o i64set/i64set.go -p i64set
//go:generate go_generics -i internal/set/set_test.go -t P=int64 -o i64set/i64set_test.go -p i64set

// NewInt64Set is a convenience function to create a new i64set.Set
func NewInt64Set() *i64set.Set {
	return i64set.New()
}

//go:generate mkdir -p uset
//go:generate go_generics -i internal/set/set.go -t T=uint -o uset/uset.go -p uset
//go:generate go_generics -i internal/set/set_test.go -t P=uint -o uset/uset_test.go -p uset

// NewUintSet is a convenience function to create a new uset.Set
func NewUintSet() *uset.Set {
	return uset.New()
}

//go:generate mkdir -p u8set
//go:generate go_generics -i internal/set/set.go -t T=uint8 -o u8set/set.go -p u8set
//go:generate go_generics -i internal/set/set_test.go -t P=uint8 -o u8set/set_test.go -p u8set

// NewUint8Set is a convenience function to create a new u8set.Set
func NewUint8Set() *u8set.Set {
	return u8set.New()
}

//go:generate mkdir -p u16set
//go:generate go_generics -i internal/set/set.go -t T=uint16 -o u16set/u16set.go -p u16set
//go:generate go_generics -i internal/set/set_test.go -t P=uint16 -o u16set/u16set_test.go -p u16set

// NewUint16Set is a convenience function to create a new u16set.Set
func NewUint16Set() *u16set.Set {
	return u16set.New()
}

//go:generate mkdir -p u32set
//go:generate go_generics -i internal/set/set.go -t T=uint32 -o u32set/u32set.go -p u32set
//go:generate go_generics -i internal/set/set_test.go -t P=uint32 -o u32set/u32set_test.go -p u32set

// NewUint32Set is a convenience function to create a new u32set.Set
func NewUint32Set() *u32set.Set {
	return u32set.New()
}

//go:generate mkdir -p u64set
//go:generate go_generics -i internal/set/set.go -t T=uint64 -o u64set/u64set.go -p u64set
//go:generate go_generics -i internal/set/set_test.go -t P=uint64 -o u64set/u64set_test.go -p u64set

// NewUint64Set is a convenience function to create a new u64set.Set
func NewUint64Set() *u64set.Set {
	return u64set.New()
}

//go:generate mkdir -p strset
//go:generate go_generics -i internal/set/set.go -t T=string -o strset/strset.go -p strset
//go:generate go_generics -i internal/set/set_test.go -t P=string -o strset/strset_test.go -p strset

// NewStringSet is a convenience function to create a new strset.Set
func NewStringSet() *strset.Set {
	return strset.New()
}
