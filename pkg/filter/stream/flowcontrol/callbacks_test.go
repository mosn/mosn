package flowcontrol

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultCallbacks(t *testing.T) {
	mockConfig := &Config{
		GlobalSwitch: false,
		Monitor:      false,
		KeyType:      "PATH",
	}
	cb := &DefaultCallbacks{config: mockConfig}
	assert.False(t, cb.Enabled())
	mockConfig.GlobalSwitch = true
	assert.True(t, cb.Enabled())
}
