package sofarpc

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/proxy"
	"golang.org/x/net/http2"
)

type codecClient struct {
	proxy.BaseCodeClient
	http2Conn http2.ClientConn
}

func (cc *codecClient) NewStream(streamId uint32, respDecoder types.StreamDecoder) types.StreamEncoder {

}
