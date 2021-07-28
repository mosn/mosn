package mirror

import (
	"context"
	"net"

	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"
)

type mirror struct {
	amplification  int
	broadcast      bool
	receiveHandler api.StreamReceiverFilterHandler
	dp             api.ProtocolName
	up             api.ProtocolName
	ctx            context.Context
	headers        api.HeaderMap
	data           buffer.IoBuffer
	trailers       api.HeaderMap
	cHeaders       api.HeaderMap
	cData          buffer.IoBuffer
	cTrailers      api.HeaderMap
	clusterName    string
	cluster        types.ClusterInfo
	sender         types.StreamSender
	host           types.Host
}

func (m *mirror) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	m.receiveHandler = handler
}

func (m *mirror) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {

	if m.receiveHandler.Route() == nil || m.receiveHandler.Route().RouteRule() == nil {
		return api.StreamFilterContinue
	}

	mirrorPolicy := m.receiveHandler.Route().RouteRule().Policy().MirrorPolicy()
	if !mirrorPolicy.IsMirror() {
		return api.StreamFilterContinue
	}

	clusterName := mirrorPolicy.ClusterName()

	utils.GoWithRecover(func() {
		clusterAdapter := cluster.GetClusterMngAdapterInstance()

		m.ctx = mosnctx.WithValue(mosnctx.Clone(ctx), types.ContextKeyBufferPoolCtx, nil)
		if headers != nil {
			// ! xprotocol should reimplement Clone function, not use default, trans protocol.CommonHeader
			h := headers.Clone()
			// nolint
			if _, ok := h.(protocol.CommonHeader); ok {
				log.DefaultLogger.Errorf("not support mirror, protocal {%v} must implement Clone function", mosnctx.Get(m.ctx, types.ContextKeyDownStreamProtocol))
				return
			}
			m.headers = h
		}
		if buf != nil {
			m.data = buf.Clone()
		}
		if trailers != nil {
			m.trailers = trailers.Clone()
		}

		m.dp, m.up = m.getProtocol()

		snap := clusterAdapter.GetClusterSnapshot(ctx, clusterName)
		if snap == nil {
			log.DefaultLogger.Errorf("mirror cluster {%s} not found", clusterName)
			return
		}
		m.cluster = snap.ClusterInfo()
		m.clusterName = clusterName

		// cover once
		m.cover()

		amplification := m.amplification
		if m.broadcast {
			amplification = 0
			for _, host := range snap.HostSet().Hosts() {
				if host.Health() {
					amplification++
				}
			}
		}

		for i := 0; i < amplification; i++ {
			connPool, host := clusterAdapter.ConnPoolForCluster(m, snap, m.up)
			if connPool == nil {
				if log.DefaultLogger.GetLogLevel() >= log.INFO {
					log.DefaultLogger.Infof("mirror get connPool failed, cluster:%s", m.clusterName)
				}
				break
			}
			var (
				streamSender types.StreamSender
				failReason   types.PoolFailureReason
			)

			if m.up == protocol.HTTP1 {
				// ! http1 use fake receiver reduce connect
				_, streamSender, failReason = connPool.NewStream(m.ctx, &receiver{})
			} else {
				_, streamSender, failReason = connPool.NewStream(m.ctx, nil)
			}

			if failReason != "" {
				m.OnFailure(failReason, host)
				continue
			}

			m.OnReady(streamSender, host)
		}
	}, nil)
	if m.broadcast {
		m.receiveHandler.SendHijackReply(api.SuccessCode, m.headers)
		return api.StreamFilterStop
	}
	return api.StreamFilterContinue
}

func (m *mirror) OnDestroy() {}

func (m *mirror) getProtocol() (dp, up types.ProtocolName) {
	dp = m.getDownStreamProtocol()
	up = m.getUpstreamProtocol()
	return
}

func (m *mirror) getDownStreamProtocol() (prot types.ProtocolName) {
	if dp, ok := mosnctx.Get(m.ctx, types.ContextKeyConfigDownStreamProtocol).(string); ok {
		return types.ProtocolName(dp)
	}
	return m.receiveHandler.RequestInfo().Protocol()
}

func (m *mirror) getUpstreamProtocol() (currentProtocol types.ProtocolName) {
	configProtocol, ok := mosnctx.Get(m.ctx, types.ContextKeyConfigUpStreamProtocol).(string)
	if !ok {
		configProtocol = string(protocol.Xprotocol)
	}

	if m.receiveHandler.Route() != nil && m.receiveHandler.Route().RouteRule() != nil && m.receiveHandler.Route().RouteRule().UpstreamProtocol() != "" {
		configProtocol = m.receiveHandler.Route().RouteRule().UpstreamProtocol()
	}

	if configProtocol == string(protocol.Auto) {
		currentProtocol = m.getDownStreamProtocol()
	} else {
		currentProtocol = types.ProtocolName(configProtocol)
	}
	return currentProtocol
}

func (m *mirror) MetadataMatchCriteria() api.MetadataMatchCriteria {
	return nil
}

func (m *mirror) DownstreamConnection() net.Conn {
	return m.receiveHandler.Connection().RawConn()
}

func (m *mirror) DownstreamHeaders() types.HeaderMap {
	return m.headers
}

func (m *mirror) DownstreamContext() context.Context {
	return m.ctx
}

func (m *mirror) DownstreamCluster() types.ClusterInfo {
	return m.cluster
}

func (m *mirror) DownstreamRoute() api.Route {
	return m.receiveHandler.Route()
}

func (m *mirror) OnFailure(reason types.PoolFailureReason, host types.Host) {}

func (m *mirror) OnReady(sender types.StreamSender, host types.Host) {
	m.sender = sender
	m.host = host

	m.sendDataOnce()
}

func (m *mirror) sendDataOnce() {
	endStream := m.data == nil && m.trailers == nil

	m.sender.AppendHeaders(m.ctx, m.cHeaders, endStream)

	if endStream {
		return
	}

	endStream = m.trailers == nil
	m.sender.AppendData(m.ctx, m.cData, endStream)

	if endStream {
		return
	}

	m.sender.AppendTrailers(m.ctx, m.cTrailers)
}

func (m *mirror) cover() {
	if m.dp == m.up {
		m.cHeaders = m.headers
		m.cData = m.data
		m.cTrailers = m.trailers
		return
	}

	m.cHeaders = m.coverHeader()
	m.cData = m.converData()
	m.cTrailers = m.convertTrailer()
}

func (m *mirror) coverHeader() types.HeaderMap {
	convHeader, err := protocol.ConvertHeader(m.ctx, m.dp, m.up, m.headers)
	if err == nil {
		return convHeader
	}
	log.Proxy.Warnf(m.ctx, "[proxy] [upstream] [mirror] convert header from %s to %s failed, %s", m.dp, m.up, err.Error())
	return m.headers
}

func (m *mirror) converData() types.IoBuffer {
	convData, err := protocol.ConvertData(m.ctx, m.dp, m.up, m.data)
	if err == nil {
		return convData
	}
	log.Proxy.Warnf(m.ctx, "[proxy] [upstream] [mirror] convert data from %s to %s failed, %s", m.dp, m.up, err.Error())
	return m.data
}

func (m *mirror) convertTrailer() types.HeaderMap {
	convTrailers, err := protocol.ConvertTrailer(m.ctx, m.dp, m.up, m.trailers)
	if err == nil {
		return convTrailers
	}
	log.Proxy.Warnf(m.ctx, "[proxy] [upstream] [mirror] convert trailers from %s to %s failed, %s", m.dp, m.up, err.Error())
	return m.trailers
}
