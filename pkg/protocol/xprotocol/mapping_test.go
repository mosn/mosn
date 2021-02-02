package xprotocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	mosnctx "mosn.io/pkg/context"
)

// type xprotocolMapping struct{}

func TestMapping(t *testing.T) {
	var ctx = context.TODO()
	var xm = xprotocolMapping{}
	// 1, sub protocol is nil
	_, err := xm.MappingHeaderStatusCode(ctx, nil)
	assert.NotNil(t, err)

	// 2. cannot get mapping
	mCtx := mosnctx.WithValue(ctx, mosnctx.ContextSubProtocol, "xxx-proto")
	_, err = xm.MappingHeaderStatusCode(mCtx, nil)
	assert.NotNil(t, err)

	// 3. normal
}
