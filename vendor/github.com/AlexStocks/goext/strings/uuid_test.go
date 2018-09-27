package gxstrings

import (
	"fmt"
	"testing"
)

func TestNewUUID(t *testing.T) {
	uuid := NewUUID()
	fmt.Println("uuid:", uuid.String())
}
