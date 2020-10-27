package bolt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/types"
)

func TestMatcher(t *testing.T) {
	var testCases = []struct {
		match []byte
		types.MatchResult
	}{
		{
			[]byte{1},
			types.MatchSuccess,
		},
		{
			[]byte{},
			types.MatchAgain,
		},
		{
			[]byte{2},
			types.MatchFailed,
		},
	}

	for _, cas := range testCases {
		assert.Equal(t, boltMatcher(cas.match), cas.MatchResult)
	}
}
