package lib

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/protocol/xprotocol/boltv2"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbothrift"
	"mosn.io/mosn/pkg/protocol/xprotocol/tars"
	xstream "mosn.io/mosn/pkg/stream/xprotocol"
	"mosn.io/mosn/pkg/trace"
	tracehttp "mosn.io/mosn/pkg/trace/sofa/http"
	xtrace "mosn.io/mosn/pkg/trace/sofa/xprotocol"
	tracebolt "mosn.io/mosn/pkg/trace/sofa/xprotocol/bolt"
	"mosn.io/mosn/test/lib/types"
)

type CreateMockServer func(config interface{}) types.MockServer

type CreateMockClient func(config interface{}) types.MockClient

// protoco: CreateFunc
var clientFactory map[string]CreateMockClient = map[string]CreateMockClient{}
var serverFactory map[string]CreateMockServer = map[string]CreateMockServer{}

func RegisterCreateClient(protocol string, f CreateMockClient) {
	clientFactory[protocol] = f
}

func RegisterCreateServer(protocol string, f CreateMockServer) {
	serverFactory[protocol] = f
}

func CreateServer(protocol string, config interface{}) types.MockServer {
	f, ok := serverFactory[protocol]
	if !ok {
		return nil
	}
	return f(config)
}

func CreateClient(protocol string, config interface{}) types.MockClient {
	f, ok := clientFactory[protocol]
	if !ok {
		return nil
	}
	return f(config)
}

// init for clients
func init() {
	// tracer driver register
	trace.RegisterDriver("SOFATracer", trace.NewDefaultDriverImpl())
	// xprotocol action register
	xprotocol.RegisterXProtocolAction(xstream.NewConnPool, xstream.NewStreamFactory, func(codec api.XProtocolCodec) {
		name := codec.ProtocolName()
		trace.RegisterTracerBuilder("SOFATracer", name, xtrace.NewTracer)
	})
	// xprotocol register
	_ = xprotocol.RegisterXProtocolCodec(&bolt.XCodec{})
	_ = xprotocol.RegisterXProtocolCodec(&boltv2.XCodec{})
	_ = xprotocol.RegisterXProtocolCodec(&dubbo.XCodec{})
	_ = xprotocol.RegisterXProtocolCodec(&dubbothrift.XCodec{})
	_ = xprotocol.RegisterXProtocolCodec(&tars.XCodec{})
	// trace register
	xtrace.RegisterDelegate(bolt.ProtocolName, tracebolt.Boltv1Delegate)
	xtrace.RegisterDelegate(boltv2.ProtocolName, tracebolt.Boltv1Delegate)
	trace.RegisterTracerBuilder("SOFATracer", protocol.HTTP1, tracehttp.NewTracer)
}
