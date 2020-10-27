package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertReasonToCode(t *testing.T) {
	reason := StreamConnectionSuccessed
	assert.Equal(t, ConvertReasonToCode(reason), SuccessCode)
}
