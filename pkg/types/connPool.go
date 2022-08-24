package types

import (
	"context"
	"mosn.io/api"
)

type RegisterCoonPool interface {
	NewConnPool(ctx context.Context, codec api.XProtocolCodec, host Host) ConnectionPool

	GetApiPoolMode() api.PoolMode
}
