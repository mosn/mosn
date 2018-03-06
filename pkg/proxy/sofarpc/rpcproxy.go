package sofarpc

import (
	"fmt"
	"reflect"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"errors"
	"gitlab.alipay-inc.com/afe/mosn/pkg/router"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
)

// 实现 sofa RPC 的 反向代理

// ReadFilter
type rpcproxy struct {
	clusterManager      types.ClusterManager
	readCallbacks       types.ReadFilterCallbacks
	upstreamConnection  types.ClientConnection
	requestInfo         types.RequestInfo
	upstreamCallbacks   UpstreamCallbacks
	downstreamCallbacks DownstreamCallbacks

	upstreamConnecting bool

	protocols    sofarpc.Protocols
	routerConfig types.RouterConfig
	clusterName  string
}

func NewRPCProxy(config *v2.RpcProxy, clusterManager types.ClusterManager) RpcProxy {
	proxy := &rpcproxy{
		clusterManager: clusterManager,
		requestInfo:    network.NewRequestInfo(),
		protocols:      sofarpc.DefaultProtocols(),
	}

	proxy.routerConfig, _ = router.CreateRouteConfig(protocol.SofaRpc, config)

	proxy.upstreamCallbacks = &upstreamCallbacks{
		proxy: proxy,
	}
	proxy.downstreamCallbacks = &downstreamCallbacks{
		proxy: proxy,
	}

	return proxy
}

type upstreamCallbacks struct {
	proxy *rpcproxy
}

////rpc onData，ADD Decode
//
//var pipelineDataChan = make(chan interface{})

func (p *rpcproxy) OnData(buf types.IoBuffer) types.FilterStatus {
	bytesRecved := p.requestInfo.BytesReceived() + uint64(buf.Len())
	p.requestInfo.SetBytesReceived(bytesRecved)

	fmt.Println("RPC MESH Receive Lens:", buf.Len())

	var out = make([]sofarpc.RpcCommand, 0, 1)

	p.protocols.Decode(nil, buf, &out)

	if len(out) > 0 {
		command := out[0]
		var headers map[string]string

		// todo: combine Decode + Handle, just provide interface to write back headers, data, trailer
		p.protocols.Handle(command.GetProtocolCode(), func(requestCommand sofarpc.BoltRequestCommand) {
			log.DefaultLogger.Println("enter in fake callback")

			headers = requestCommand.GetRequestHeader()
		}, command)

		if headers == nil {
			return types.StopIteration
		}

		//do some route by service name
		route := p.routerConfig.Route(headers)

		if route == nil || route.RouteRule() == nil {
			// no route
			p.onDataErr()

			return types.StopIteration
		}

		if err := p.initializeUpstreamConnection(route.RouteRule().ClusterName()); err != nil {
			p.onDataErr()
		} else {
			//send data after decode finished
			p.upstreamConnection.Write(buf)
		}
	}

	return types.StopIteration
}

func (p *rpcproxy) onDataErr() {
	// todo: discard request buf
	// todo: close connection
}

//rpc upstream onEvent
func (uc *upstreamCallbacks) OnEvent(event types.ConnectionEvent) {
	uc.proxy.onUpstreamEvent(event)
}

func (uc *upstreamCallbacks) OnAboveWriteBufferHighWatermark() {
	// TODO
}

func (uc *upstreamCallbacks) OnBelowWriteBufferLowWatermark() {
	// TODO
}

func (p *rpcproxy) onUpstreamData(buffer types.IoBuffer) {
	bytesSent := p.requestInfo.BytesSent() + uint64(buffer.Len())
	p.requestInfo.SetBytesSent(bytesSent)

	p.readCallbacks.Connection().Write(buffer)
}

func (uc *upstreamCallbacks) OnData(buffer types.IoBuffer) types.FilterStatus {
	uc.proxy.onUpstreamData(buffer)

	return types.StopIteration
}

func (uc *upstreamCallbacks) OnNewConnection() types.FilterStatus {
	return types.Continue
}

func (uc *upstreamCallbacks) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {}

//rpc realize upstream on event

func (p *rpcproxy) onUpstreamEvent(event types.ConnectionEvent) {
	switch event {
	case types.RemoteClose:
		// TODO: inc remote failed stat
		if p.upstreamConnecting {
			p.requestInfo.SetResponseFlag(types.UpstreamConnectionFailure)
		} else {
			p.readCallbacks.Connection().Close(types.FlushWrite, types.LocalClose)
		}
	case types.LocalClose:
		// TODO: inc local failed stat
	case types.OnConnect:
		p.upstreamConnecting = true
	case types.Connected:
		p.upstreamConnecting = false

		p.onConnectionSuccess()
	case types.ConnectTimeout:
		p.requestInfo.SetResponseFlag(types.UpstreamConnectionFailure)
	}
}

func (p *rpcproxy) onDownstreamEvent(event types.ConnectionEvent) {
	if p.upstreamConnecting {
		if event == types.RemoteClose {
			p.upstreamConnection.Close(types.FlushWrite, types.LocalClose)
		} else if event == types.LocalClose {
			p.upstreamConnection.Close(types.NoFlush, types.LocalClose)
		}
	}
}

func (p *rpcproxy) ReadDisableUpstream(disable bool) {
	// TODO
}

func (p *rpcproxy) ReadDisableDownstream(disable bool) {
	// TODO
}

func (p *rpcproxy) closeUpstreamConnection() {
	// TODO: finalize upstream connection stats
	p.upstreamConnection.Close(types.NoFlush, types.LocalClose)
}

func (p *rpcproxy) initializeUpstreamConnection(clusterName string) error {
	clusterSnapshot := p.clusterManager.Get(clusterName, nil)

	if reflect.ValueOf(clusterSnapshot).IsNil() {
		p.requestInfo.SetResponseFlag(types.NoRouteFound)
		p.onInitFailure(NoRoute)

		return errors.New(fmt.Sprintf("unkown cluster %s", clusterName))
	}

	clusterInfo := clusterSnapshot.ClusterInfo()
	clusterConnectionResource := clusterInfo.ResourceManager().ConnectionResource()

	if !clusterConnectionResource.CanCreate() {
		p.requestInfo.SetResponseFlag(types.UpstreamOverflow)
		p.onInitFailure(ResourceLimitExceeded)

		return errors.New(fmt.Sprintf("upstream overflow in cluster %s", clusterName))
	}

	connectionData := p.clusterManager.TcpConnForCluster(clusterName, nil)

	if connectionData.Connection == nil {
		p.requestInfo.SetResponseFlag(types.NoHealthyUpstream)
		p.onInitFailure(NoHealthyUpstream)

		return errors.New(fmt.Sprintf("no healthy upstream in cluster %s", clusterName))
	}

	p.readCallbacks.SetUpstreamHost(connectionData.HostInfo)
	clusterConnectionResource.Increase()

	upstreamConnection := connectionData.Connection
	upstreamConnection.AddConnectionCallbacks(p.upstreamCallbacks)
	upstreamConnection.FilterManager().AddReadFilter(p.upstreamCallbacks)

	if err := upstreamConnection.Connect(); err != nil {
		return err
	}

	upstreamConnection.SetNoDelay(true)

	p.upstreamConnection = upstreamConnection
	p.requestInfo.OnUpstreamHostSelected(connectionData.HostInfo)

	// TODO: update upstream stats

	return nil
}

func (p *rpcproxy) onConnectionSuccess() {}

func (p *rpcproxy) onInitFailure(reason UpstreamFailureReason) {
	p.readCallbacks.Connection().Close(types.NoFlush, types.LocalClose)
}

func (p *rpcproxy) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {
	p.readCallbacks = cb

	p.readCallbacks.Connection().AddConnectionCallbacks(p.downstreamCallbacks)

	p.requestInfo.SetDownstreamLocalAddress(p.readCallbacks.Connection().LocalAddr())
	p.requestInfo.SetDownstreamRemoteAddress(p.readCallbacks.Connection().RemoteAddr())

	// TODO: set downstream connection stats
}

func (p *rpcproxy) OnNewConnection() types.FilterStatus {
	return types.Continue
}

// ConnectionCallbacks
type downstreamCallbacks struct {
	proxy *rpcproxy
}

func (dc *downstreamCallbacks) OnEvent(event types.ConnectionEvent) {
	dc.proxy.onDownstreamEvent(event)
}

func (dc *downstreamCallbacks) OnAboveWriteBufferHighWatermark() {
	// TODO
}

func (dc *downstreamCallbacks) OnBelowWriteBufferLowWatermark() {
	// TODO
}
