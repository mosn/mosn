// Copyright 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// package gxarray provide array/slice related algorithms
package gxarray

func RemoveElem(array []string, element string) ([]string, bool) {
	for i := range array {
		if array[i] == element {
			array = append(array[:i], array[i+1:]...)
			return array, true
		}
	}

	return array, false
}
