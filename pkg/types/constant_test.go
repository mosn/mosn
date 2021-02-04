package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
)

func TestConvertReasonToCode(t *testing.T) {
	reason := StreamConnectionSuccessed
	assert.Equal(t, ConvertReasonToCode(reason), api.SuccessCode)
}
