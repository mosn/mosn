// Copyright 2016 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// 2016/10/01
// Package gxbitmap implements bitmap in golang.

package gxbitmap

import "fmt"

type Bitmap struct {
	data []byte
	size int // bit numbers
}

func NewBitmap(size int) Bitmap {
	if size <= 0 {
		panic("NewBitmap(@size <= 0)")
	}
	var s = size >> 3
	if (size % 8) != 0 {
		s++
	}

	return Bitmap{data: make([]byte, s), size: s << 3}
}

func (b *Bitmap) Set(pos int) error {
	if pos > b.size || pos < 0 {
		return fmt.Errorf("@pos{%d}, b.size{%d}", pos, b.size)
	}

	b.data[pos>>3] |= 0x01 << (uint(pos) & 0x07)

	return nil
}

func (b *Bitmap) Clear(pos int) error {
	if pos > b.size || pos < 0 {
		return fmt.Errorf("@pos{%d}, b.size{%d}", pos, b.size)
	}

	b.data[pos>>3] &^= 0x01 << (uint(pos) & 0x07)

	return nil
}

func (b *Bitmap) Get(pos int) int {
	if pos > b.size || pos < 0 {
		return 0
	}

	return int((b.data[pos>>3] >> (uint(pos) & 0x07)) & 0x01)
}
