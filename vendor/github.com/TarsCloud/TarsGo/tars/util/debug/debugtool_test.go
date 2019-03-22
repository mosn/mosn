package debugutil

import (
	"testing"
)

func TestDumpStack(t *testing.T) {
	DumpStack(true, "testdump")
}
