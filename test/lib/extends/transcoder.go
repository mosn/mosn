package extends

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
)

var transMapping = map[api.ProtocolName]map[api.ProtocolName]string{
	protocol.HTTP1: map[api.ProtocolName]string{
		protocol.HTTP2: "httpTohttp2",
	},
	protocol.HTTP2: map[api.ProtocolName]string{
		protocol.HTTP1: "http2Tohttp",
	},
}

func GetTransFilter(original, trans api.ProtocolName) (in, mid string) {
	in = transMapping[original][trans]
	mid = transMapping[trans][original]
	return in, mid
}
