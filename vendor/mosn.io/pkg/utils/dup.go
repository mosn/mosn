// +build linux,!arm64,!amd64,!386 darwin,!amd64,!386 windows

package utils

import (
	"fmt"
	"runtime"
)

func Dup(from, to int) error {
	fmt.Printf("GOOS %s, GOARCH %s is not support dup\n", runtime.GOOS, runtime.GOARCH)
	return nil
}
