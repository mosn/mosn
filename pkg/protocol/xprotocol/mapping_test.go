package xprotocol

import (
	"context"
	"github.com/stretchr/testify/assert"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/types"
	"testing"
)

// type xprotocolMapping struct{}

func TestMapping(t *testing.T) {
	var ctx = context.TODO()
	var xm = xprotocolMapping{}
	// 1, sub protocol is nil
	_, err := xm.MappingHeaderStatusCode(ctx, nil)
	assert.NotNil(t, err)

	// 2. cannot get mapping
	mCtx := mosnctx.WithValue(ctx, types.ContextSubProtocol, "xxx-proto")
	_, err = xm.MappingHeaderStatusCode(mCtx, nil)
	assert.NotNil(t, err)

	// 3. normal
}
