/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package proxy

import (
	"bufio"
	"errors"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	mosnhttp "mosn.io/mosn/pkg/protocol/http"
	httpstream "mosn.io/mosn/pkg/stream/http"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"

	"github.com/valyala/fasthttp"
)

type websocketProxy struct {
	ds                  *downStream
	req                 *fasthttp.Request
	readCallbacks       api.ReadFilterCallbacks
	requestInfo         types.RequestInfo
	clusterManager      types.ClusterManager
	clientReadBuffer    *bufio.Reader
	upstreamCallbacks   UpstreamCallbacks
	upstreamConnection  types.ClientConnection
	downstreamCallbacks DownstreamCallbacks
}

func createWebsocketProxy(ds *downStream, args *types.ProxyWebsocketArgs) *websocketProxy {
	proxy := &websocketProxy{
		req:              &fasthttp.Request{},
		ds:               ds,
		clientReadBuffer: args.CurrentBufferReader,
		requestInfo:      ds.requestInfo,
		clusterManager:   cluster.GetClusterMngAdapterInstance().ClusterManager,
		readCallbacks:    ds.proxy.readCallbacks,
	}
	proxy.upstreamCallbacks = &upstreamCallbacks{
		proxy: proxy,
	}
	proxy.downstreamCallbacks = &websocketDownstreamCallbacks{
		proxy: proxy,
	}
	proxy.readCallbacks.Connection().SetReadDisable(true)
	proxy.readCallbacks.Connection().AddConnectionEventListener(proxy.downstreamCallbacks)
	proxy.requestInfo.SetDownstreamRemoteAddress(proxy.readCallbacks.Connection().RemoteAddr())
	proxy.requestInfo.SetDownstreamLocalAddress(proxy.readCallbacks.Connection().LocalAddr())

	return proxy
}

// DownstreamCallbacks for downstream's callbacks
type DownstreamCallbacks interface {
	api.ConnectionEventListener
}

// ConnectionEventListener
type websocketDownstreamCallbacks struct {
	proxy *websocketProxy
}

func (dc *websocketDownstreamCallbacks) OnEvent(event api.ConnectionEvent) {
	dc.proxy.onDownstreamEvent(event)
}

func (uc *websocketDownstreamCallbacks) OnData(buffer buffer.IoBuffer) api.FilterStatus {
	return uc.proxy.OnData(buffer)
}

// ConnectionEventListener
// ReadFilter
type upstreamCallbacks struct {
	proxy *websocketProxy
}

func (uc *upstreamCallbacks) OnData(buffer buffer.IoBuffer) api.FilterStatus {
	uc.proxy.onUpstreamData(buffer)
	return api.Stop
}

func (uc *upstreamCallbacks) OnNewConnection() api.FilterStatus {
	return api.Continue
}

func (uc *upstreamCallbacks) InitializeReadFilterCallbacks(cb api.ReadFilterCallbacks) {}

func (uc *upstreamCallbacks) OnEvent(event api.ConnectionEvent) {
	uc.proxy.onUpstreamEvent(event)
}

func (wp *websocketProxy) OnData(buffer buffer.IoBuffer) api.FilterStatus {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(wp.ds.context, "[proxy] [websocket] [ondata] read data , len = %v", buffer.Len())
	}
	bytesRecved := wp.requestInfo.BytesReceived() + uint64(buffer.Len())
	wp.requestInfo.SetBytesReceived(bytesRecved)

	wp.upstreamConnection.Write(buffer.Clone())
	buffer.Drain(buffer.Len())
	return api.Stop
}

func (wp *websocketProxy) onDownstreamEvent(event api.ConnectionEvent) {
	if wp.upstreamConnection != nil {
		if event == api.RemoteClose {
			wp.upstreamConnection.Close(api.FlushWrite, api.LocalClose)
		} else if event == api.LocalClose {
			wp.upstreamConnection.Close(api.NoFlush, api.LocalClose)
		}
	}
}

func (wp *websocketProxy) finalizeUpstreamConnectionStats() {
	hostInfo := wp.readCallbacks.UpstreamHost()
	if host, ok := hostInfo.(types.Host); ok {
		host.ClusterInfo().ResourceManager().Connections().Decrease()
		host.ClusterInfo().Stats().UpstreamConnectionActive.Dec(1)
		host.HostStats().UpstreamConnectionActive.Dec(1)
	}
}

func (wp *websocketProxy) closeDownstreamConnection(ccType api.ConnectionCloseType, eventType api.ConnectionEvent) {
	wp.ds.cleanStream()
	wp.readCallbacks.Connection().Close(ccType, eventType)
}

func (wp *websocketProxy) onUpstreamEvent(event api.ConnectionEvent) {
	switch event {
	case api.RemoteClose,api.OnWriteErrClose,api.OnWriteTimeout:
		wp.finalizeUpstreamConnectionStats()
		wp.closeDownstreamConnection(api.FlushWrite, api.RemoteClose)
	case api.LocalClose, api.OnReadErrClose:
		wp.finalizeUpstreamConnectionStats()
		wp.closeDownstreamConnection(api.NoFlush, api.LocalClose)

	case api.OnConnect:
	case api.Connected:
		wp.upstreamConnection.SetNoDelay(true)
		wp.readCallbacks.Connection().SetReadDisable(false)
	case api.ConnectTimeout:
		wp.finalizeUpstreamConnectionStats()

		wp.requestInfo.SetResponseFlag(api.UpstreamConnectionFailure)
		wp.upstreamConnection.Close(api.NoFlush, api.LocalClose)
	case api.ConnectFailed:
		wp.requestInfo.SetResponseFlag(api.UpstreamConnectionFailure)
	}
}

func (wp *websocketProxy) onUpstreamData(buffer types.IoBuffer) {
	bytesSent := wp.requestInfo.BytesSent() + uint64(buffer.Len())
	wp.requestInfo.SetBytesSent(bytesSent)

	wp.readCallbacks.Connection().Write(buffer.Clone())
	buffer.Drain(buffer.Len())
}

func (wp *websocketProxy) onInitFailure(reason UpstreamFailureReason) {
	wp.ds.requestInfo.SetResponseFlag(api.NoHealthyUpstream)
	wp.ds.sendHijackReply(api.NoHealthUpstreamCode, wp.ds.downstreamReqHeaders)
	endStream := true
	wp.ds.downstreamReqHeaders.Set("Connection", "Close")
	wp.ds.appendHeaders(endStream)
}

func (wp *websocketProxy) initUpstreamConnection() error {
	retryTimes := wp.ds.snapshot.HostNum(nil)
	retryRemaining := wp.ds.retryState.retiesRemaining
	if retryRemaining > 0 && int(retryRemaining) < retryTimes {
		retryTimes = int(retryRemaining)
	}

	for i := 0; i < retryTimes; i++ {
		if wp.ds.upstreamRequest == nil || i > 0 {
			wp.ds.chooseHost(wp.ds.downstreamReqDataBuf == nil && wp.ds.downstreamReqTrailers == nil)
			if wp.ds.upstreamRequest == nil {
				return errors.New("no healthy upstream")
			}
		}

		upstreamHost := wp.ds.upstreamRequest.connPool.Host()
		connectionData := upstreamHost.CreateConnection(wp.ds.context)
		if connectionData.Connection == nil {
			wp.requestInfo.SetResponseFlag(api.NoHealthyUpstream)
			return errors.New("no healthy upstream")
		}
		wp.readCallbacks.SetUpstreamHost(connectionData.Host)

		clusterConnectionResource := wp.ds.snapshot.ClusterInfo().ResourceManager().Connections()
		clusterConnectionResource.Increase()

		upstreamConnection := connectionData.Connection
		wp.upstreamConnection = upstreamConnection
		wp.upstreamConnection.AddConnectionEventListener(wp.upstreamCallbacks)
		wp.upstreamConnection.FilterManager().AddReadFilter(wp.upstreamCallbacks)
		wp.upstreamConnection.SetReadDisable(true)

		if err := upstreamConnection.Connect(); err != nil {
			log.Proxy.Debugf(wp.ds.context, "[proxy] [websocket] connect to upstream failed")
			wp.requestInfo.SetResponseFlag(api.NoHealthyUpstream)
			continue
		}

		connectionData.Host.HostStats().UpstreamConnectionActive.Inc(1)
		connectionData.Host.ClusterInfo().Stats().UpstreamConnectionActive.Inc(1)

		wp.requestInfo.OnUpstreamHostSelected(connectionData.Host)
		wp.requestInfo.SetUpstreamLocalAddress(connectionData.Host.AddressString())

		goto Success
	}

	return errors.New("no healthy upstream")

Success:
	return nil
}

func (wp *websocketProxy) websocketHandshake() error {
	headers := wp.ds.downstreamReqHeaders.(mosnhttp.RequestHeader)

	// clear 'Connection:close' header for keepalive connection with upstream
	if headers.ConnectionClose() {
		headers.Del("Connection")
	}

	httpstream.FillRequestHeadersFromCtxVar(wp.ds.context, headers, wp.ds.DownstreamConnection().RemoteAddr())

	headers.CopyTo(&wp.req.Header)
	if _, err := wp.req.WriteTo(wp.upstreamConnection.RawConn()); err != nil {
		return err
	}

	bbr := bufio.NewReader(wp.upstreamConnection.RawConn())
	rsp := fasthttp.AcquireResponse()
	err := rsp.Read(bbr)
	if err != nil {
		return err
	}

	wp.requestInfo.SetResponseCode(rsp.StatusCode())

	var bytesSent int64
	bytesSent, err = rsp.WriteTo(wp.readCallbacks.Connection().RawConn())
	wp.requestInfo.SetBytesSent(uint64(bytesSent) + wp.requestInfo.BytesSent() )

	if err == nil {
		log.Proxy.Debugf(wp.ds.context, "[proxy] [websocket] get upstream response ok")
		wp.upstreamConnection.SetReadDisable(false)
	}

	return err
}

func (wp *websocketProxy) handleWebsocket() {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(wp.ds.context, "[proxy] [websocket] handleWebsocket findBackend, id = %d", wp.ds.ID)
	}

	if err := wp.initUpstreamConnection(); err != nil {
		log.Proxy.Errorf(wp.ds.context, "[proxy] [websocket] handleWebsocket findBackend error: %s", err.Error())
		wp.onInitFailure(NoHealthyUpstream)
		return
	}

	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(wp.ds.context, "[proxy] [websocket] handleWebsocket websocketHandshake, id = %d", wp.ds.ID)
	}

	if err := wp.websocketHandshake(); err != nil {
		log.Proxy.Errorf(wp.ds.context, "[proxy] [websocket] handleWebsocket websocketHandshake error: %s, id = %d", err.Error(), wp.ds.ID)
		wp.onInitFailure(NoHealthyUpstream)
		return
	}

	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(wp.ds.context, "[proxy] [websocket] handleWebsocket websocketDataTransfer, proxyId = %d  ", wp.ds.ID)
	}

	wp.websocketDataTransfer()
}

func (wp *websocketProxy) websocketDataTransfer() {
	utils.GoWithRecover(func() {
		bytesSent, err := wp.clientReadBuffer.WriteTo(wp.upstreamConnection.RawConn())
		wp.requestInfo.SetBytesReceived(wp.requestInfo.BytesReceived()+uint64(bytesSent))
		if err != nil {
			log.Proxy.Infof(wp.ds.context, "[proxy] [websocket] websocketDataTransfer proxy to backend error: %s, id = %d", err.Error(), wp.ds.ID)

		}
	}, nil)
}
