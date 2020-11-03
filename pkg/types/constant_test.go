package types

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConvertReasonToCode(t *testing.T) {
	reason := StreamConnectionSuccessed
	assert.Equal(t, ConvertReasonToCode(reason), SuccessCode)
}