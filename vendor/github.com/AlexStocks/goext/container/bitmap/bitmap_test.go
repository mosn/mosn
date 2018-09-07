package gxbitmap

import (
	"fmt"
	"testing"
)

var (
	bitmap Bitmap = NewBitmap(100)
)

func test(pos int) {
	bitmap.Set(pos)
	fmt.Println(bitmap.Get(pos))
	bitmap.Clear(pos)
	fmt.Println(bitmap.Get(pos))
}

func TestBitmap_Set(t *testing.T) {
	test(0)
	test(1)
	test(53)
	test(99)
	test(100)
}
