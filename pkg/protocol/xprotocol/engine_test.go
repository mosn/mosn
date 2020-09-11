package xprotocol

import (
	"context"
	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"testing"
)

func TestEngine(t *testing.T) {
	var mockProto = &mockProtocol{}
	err := RegisterProtocol("engine-proto", mockProto)
	assert.Nil(t, err)

	err = RegisterMatcher("engine-proto", mockMatcher)
	assert.Nil(t, err)

	matcher := GetMatcher("engine-proto")
	assert.NotNil(t, matcher)

	var protocols = []string{"engine-proto"}
	xEngine, err := NewXEngine(protocols)
	assert.Nil(t, err)
	assert.NotNil(t, xEngine)

	_, res := xEngine.Match(context.TODO(), buffer.NewIoBuffer(10))
	assert.Equal(t, res, types.MatchSuccess)
}
