package wasmer

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDebugParseDwarf(t *testing.T) {
	bytes, err := ioutil.ReadFile("./testdata/data.wasm")
	assert.Nil(t, err)

	debug := parseDwarf(bytes)
	assert.NotNil(t, debug)
	assert.NotNil(t, debug.data)
	assert.Equal(t, debug.codeSectionOffset, 0x326) // code section start addr

	lr := debug.getLineReader()
	assert.NotNil(t, lr)

	line := debug.SeekPC(uint64(0x2ef1)) // f3
	assert.NotNil(t, line)
}
