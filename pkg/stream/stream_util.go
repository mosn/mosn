package stream

/*
import (
	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
	"sync/atomic"
	"context"
)

// StreamSender Create a client stream and call's by proxy
func StreamSender(ctx context.Context,p types.ConnectionPool,
	receiver types.StreamReceiveListener) (types.PoolFailureReason, types.Host, types.StreamSender) {
	host := p.Host()

	subProtocol := getSubProtocol(ctx)
	c, reason := p.GetActiveClient(ctx, subProtocol)
	if c == nil {
		return reason, host, nil
	}

	if useAsyncConnect(p.Protocol(), subProtocol) && atomic.LoadUint32(&c.state) != Connected {
		return types.ConnectionFailure, host, nil
	}

	if !host.ClusterInfo().ResourceManager().Requests().CanCreate() {
		host.HostStats().UpstreamRequestPendingOverflow.Inc(1)
		host.ClusterInfo().Stats().UpstreamRequestPendingOverflow.Inc(1)
		return types.Overflow, host, nil

	}

	if p.shouldMultiplex(subProtocol) {
		atomic.AddUint64(&c.totalStream, 1)
	}

	host.HostStats().UpstreamRequestTotal.Inc(1)
	host.ClusterInfo().Stats().UpstreamRequestTotal.Inc(1)

	var streamEncoder = c.codecClient.StreamSender(ctx, receiver)

	// FIXME one way
	// is there any need to skip the metrics?
	if receiver == nil {
		return "", host, streamEncoder
	}

	streamEncoder.GetStream().AddEventListener(c)

	host.HostStats().UpstreamRequestActive.Inc(1)
	host.ClusterInfo().Stats().UpstreamRequestActive.Inc(1)
	host.ClusterInfo().ResourceManager().Requests().Increase()

	return "", host, streamEncoder
}

func getSubProtocol(ctx context.Context) types.ProtocolName {
	if ctx != nil {
		if val := mosnctx.Get(ctx, types.ContextSubProtocol); val != nil {
			if code, ok := val.(string); ok {
				return types.ProtocolName(code)
			}
		}
	}
	return ""
}

// 1. default use async connect
// 2. the pool mode was set to multiplex
func useAsyncConnect(proto api.Protocol, subproto types.ProtocolName) bool {
	// HACK LOGIC
	// if the pool mode was set to mutiplex
	// we should not use async connect
	if proto == protocol.Xprotocol &&
		xprotocol.GetProtocol(subproto).PoolMode() == types.Multiplex {
		return true
	}

	return false
}
 */
