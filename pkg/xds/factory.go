package xds

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	jsoniter "github.com/json-iterator/go"
	mv2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
	v2conv "mosn.io/mosn/pkg/xds/v2/conv"
	v3conv "mosn.io/mosn/pkg/xds/v3/conv"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func duration2String(duration *duration.Duration) string {
	d := time.Duration(duration.Seconds)*time.Second + time.Duration(duration.Nanos)*time.Nanosecond
	x := fmt.Sprintf("%.9f", d.Seconds())
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, "000")
	return x + "s"
}

// Client XdsClient
type Client interface {
	Start(config *mv2.MOSNConfig) error
	Stop()
}

// NewClient build xds Client
func NewClient() Client {
	switch types.XdsVersion {
	case types.XdsVersionV3:
		return &clientv3{}
	default:
		return &clientv2{}
	}
}

// InitStats init stats
func InitStats() {
	switch types.XdsVersion {
	case types.XdsVersionV3:
		v3conv.InitStats()
	default:
		v2conv.InitStats()
	}
}

// GetStats return xdsStats
func GetStats() types.XdsStats {
	switch types.XdsVersion {
	case types.XdsVersionV3:
		return v3conv.Stats
	default:
		return v2conv.Stats
	}
}
