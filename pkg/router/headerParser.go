package router

import "gitlab.alipay-inc.com/afe/mosn/pkg/types"

type  HeaderParser struct {
	headersToAdd    []types.Pair
	headersToRemove []*LowerCaseString
}