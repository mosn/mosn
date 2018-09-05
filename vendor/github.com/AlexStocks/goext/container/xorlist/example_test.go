// Copyright 2015 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

package gxxorlist

import (
	"fmt"
)

func Example() {
	// Create a new list and put some numbers in it.
	l := New()
	e4 := l.PushBack(4)
	e1 := l.PushFront(1)
	l.InsertBefore(3, e4)
	l.InsertAfter(2, e1)

	// Iterate through list and print its contents.
	for e, p := l.Front(); e != nil; e, p = e.Next(p), e {
		fmt.Println(e.Value)
	}

	// Output:
	// 1
	// 2
	// 3
	// 4
}
