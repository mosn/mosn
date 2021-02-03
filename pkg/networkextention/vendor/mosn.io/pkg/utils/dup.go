// +build !arm64

package utils

import "syscall"

func Dup(from, to int) error {
	return syscall.Dup2(from, to)
}
