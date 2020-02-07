package utils

import "syscall"

// if oldd ≠ newd and flags == 0, the behavior is identical to dup2(oldd, newd).
// arm support dup3 only
func Dup(from, to int) error {
	return syscall.Dup3(from, to, 0)
}
