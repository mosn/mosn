package framework

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Assert wrapps fucntions in testify/require
type Assert func(t *testing.T, actual interface{}, expected ...interface{})

var (
	Equal Assert = func(t *testing.T, actual interface{}, expected ...interface{}) {
		require.EqualValues(t, expected[0], actual)
	}
	NotEqual = func(t *testing.T, actual interface{}, expected ...interface{}) {
		require.NotEqual(t, expected[0], actual)
	}
	NotNil = func(t *testing.T, actual interface{}, expected ...interface{}) {
		require.NotNil(t, actual)
	}
	Less = func(t *testing.T, actual interface{}, expected ...interface{}) {
		require.Less(t, expected[0], actual)
	}
	LessOrEqual = func(t *testing.T, actual interface{}, expected ...interface{}) {
		require.LessOrEqual(t, expected[0], actual)
	}
	Greater = func(t *testing.T, actual interface{}, expected ...interface{}) {
		require.Greater(t, expected[0], actual)
	}
	GreaterOrEqual = func(t *testing.T, actual interface{}, expected ...interface{}) {
		require.GreaterOrEqual(t, expected[0], actual)
	}

	// TODO: Add More
)
