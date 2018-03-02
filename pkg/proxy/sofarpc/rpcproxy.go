package sofarpc



import (
	"bytes"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"fmt"
	"reflect"
)

// 实现 sofa RPC 的 四层转发

type rpcproxy struct {
	config              ProxyConfig
	clusterManager      types.ClusterManager
	readCallbacks       types.ReadFilterCallbacks
	upstreamConnection  types.ClientConnection
	requestInfo         types.RequestInfo
	upstreamCallbacks   UpstreamCallbacks
	downstreamCallbacks DownstreamCallbacks

	upstreamConnecting bool
}

