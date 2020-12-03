package util

import "unsafe"

// SliceHeader is a safe version of SliceHeader used within this project.
type SliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

// StringHeader is a safe version of StringHeader used within this project.
type StringHeader struct {
	Data unsafe.Pointer
	Len  int
}
