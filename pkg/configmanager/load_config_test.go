package configmanager

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYAMLConfigLoad(t *testing.T) {
	tests := []string{
		"config.yaml",
		"envoy.yaml",
	}

	for _, filename := range tests {
		c := YAMLConfigLoad(filepath.Join("testdata", filename))
		assert.NotNil(t, c)
	}
}
