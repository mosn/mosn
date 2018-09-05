// unit test of gxrand

package gxrand

import (
	"fmt"
	"testing"
)

func TestRandString(t *testing.T) {
	fmt.Println(RandString(4))
	fmt.Println(RandDigitString(4))
}

func TestNewUUID(t *testing.T) {
	fmt.Println(NewUUID())
}
