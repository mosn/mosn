package bolt

import (
	"github.com/stretchr/testify/assert"
	"mosn.io/api"

	"testing"
)

func TestMatcher(t *testing.T) {
	var testCases = []struct {
		match []byte
		api.MatchResult
	}{
		{
			[]byte{1},
			api.MatchSuccess,
		},
		{
			[]byte{},
			api.MatchAgain,
		},
		{
			[]byte{2},
			api.MatchFailed,
		},
	}

	for _, cas := range testCases {
		assert.Equal(t, boltMatcher(cas.match), cas.MatchResult)
	}
}
