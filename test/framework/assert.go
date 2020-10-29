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
		require.Less(t, actual, expected[0])
	}
	LessOrEqual = func(t *testing.T, actual interface{}, expected ...interface{}) {
		require.LessOrEqual(t, actual, expected[0])
	}
	Greater = func(t *testing.T, actual interface{}, expected ...interface{}) {
		require.Greater(t, actual, expected[0])
	}
	GreaterOrEqual = func(t *testing.T, actual interface{}, expected ...interface{}) {
		require.GreaterOrEqual(t, actual, expected[0])
	}

	// TODO: Add More
)
