package faultinject

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type FaultInjecter interface {
	types.ReadFilter
}