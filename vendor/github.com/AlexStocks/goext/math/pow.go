// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// gxmath provides some pow likely functions
package gxmath

func IsPowerOf2(x int) bool {
	return (x != 0) && ((x & (x - 1)) == 0)
}
