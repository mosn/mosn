package types

import (
	"github.com/stretchr/testify/assert"
	pkgtypes "mosn.io/pkg/types"

	"testing"
)

func TestConvertReasonToCode(t *testing.T) {
	reason := StreamConnectionSuccessed
	assert.Equal(t, ConvertReasonToCode(reason), pkgtypes.SuccessCode)
}
