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

package tcpproxy

import (
	"context"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// ReadFilter
type proxy struct {
	config              ProxyConfig
	clusterManager      types.ClusterManager
	readCallbacks       types.ReadFilterCallbacks
	upstreamConnection  types.ClientConnection
	requestInfo         types.RequestInfo
	upstreamCallbacks   UpstreamCallbacks
	downstreamCallbacks DownstreamCallbacks

	upstreamConnecting bool

	accessLogs []types.AccessLog
}

func NewProxy(ctx context.Context, config *v2.TCPProxy, clusterManager types.ClusterManager) Proxy {
	p := &proxy{
		config:         NewProxyConfig(config),
		clusterManager: clusterManager,
		requestInfo:    network.NewRequestInfo(),
		accessLogs:     ctx.Value(types.ContextKeyAccessLogs).([]types.AccessLog),
	}

	p.upstreamCallbacks = &upstreamCallbacks{
		proxy: p,
	}
	p.downstreamCallbacks = &downstreamCallbacks{
		proxy: p,
	}

	return p
}

func (p *proxy) OnData(buffer types.IoBuffer) types.FilterStatus {
	log.DefaultLogger.Tracef("Tcp Proxy :: read data , len = %v", buffer.Len())
	bytesRecved := p.requestInfo.BytesReceived() + uint64(buffer.Len())
	p.requestInfo.SetBytesReceived(bytesRecved)

	p.upstreamConnection.Write(buffer.Clone())
	buffer.Drain(buffer.Len())
	return types.Stop
}

func (p *proxy) OnNewConnection() types.FilterStatus {
	log.DefaultLogger.Tracef("Tcp Proxy :: accept new connection")
	return p.initializeUpstreamConnection()
}

func (p *proxy) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {
	p.readCallbacks = cb

	p.readCallbacks.Connection().AddConnectionEventListener(p.downstreamCallbacks)
	p.requestInfo.SetDownstreamRemoteAddress(p.readCallbacks.Connection().RemoteAddr())
	p.requestInfo.SetDownstreamLocalAddress(p.readCallbacks.Connection().LocalAddr())

	p.readCallbacks.Connection().SetReadDisable(true)

	// TODO: set downstream connection stats
}

func (p *proxy) initializeUpstreamConnection() types.FilterStatus {
	clusterName := p.getUpstreamCluster()

	clusterSnapshot := p.clusterManager.GetClusterSnapshot(context.Background(), clusterName)

	if reflect.ValueOf(clusterSnapshot).IsNil() {
		p.requestInfo.SetResponseFlag(types.NoRouteFound)
		p.onInitFailure(NoRoute)

		return types.Stop
	}

	clusterInfo := clusterSnapshot.ClusterInfo()
	clusterConnectionResource := clusterInfo.ResourceManager().Connections()

	if !clusterConnectionResource.CanCreate() {
		p.requestInfo.SetResponseFlag(types.UpstreamOverflow)
		p.onInitFailure(ResourceLimitExceeded)

		return types.Stop
	}

	ctx := &LbContext{
		conn: p.readCallbacks,
	}
	connectionData := p.clusterManager.TCPConnForCluster(ctx, clusterSnapshot)
	if connectionData.Connection == nil {
		p.requestInfo.SetResponseFlag(types.NoHealthyUpstream)
		p.onInitFailure(NoHealthyUpstream)

		return types.Stop
	}
	p.readCallbacks.SetUpstreamHost(connectionData.HostInfo)
	clusterConnectionResource.Increase()
	upstreamConnection := connectionData.Connection
	upstreamConnection.AddConnectionEventListener(p.upstreamCallbacks)
	upstreamConnection.FilterManager().AddReadFilter(p.upstreamCallbacks)
	p.upstreamConnection = upstreamConnection
	if err := upstreamConnection.Connect(true); err != nil {
		p.requestInfo.SetResponseFlag(types.NoHealthyUpstream)
		p.onInitFailure(NoHealthyUpstream)
		return types.Stop
	}

	p.requestInfo.OnUpstreamHostSelected(connectionData.HostInfo)
	p.requestInfo.SetUpstreamLocalAddress(upstreamConnection.LocalAddr())
	// TODO: snapshot lifecycle
	p.clusterManager.PutClusterSnapshot(clusterSnapshot)

	// TODO: update upstream stats

	return types.Continue
}

func (p *proxy) closeUpstreamConnection() {
	// TODO: finalize upstream connection stats
	p.upstreamConnection.Close(types.NoFlush, types.LocalClose)
}

func (p *proxy) getUpstreamCluster() string {
	downstreamConnection := p.readCallbacks.Connection()

	return p.config.GetRouteFromEntries(downstreamConnection)
}

func (p *proxy) onInitFailure(reason UpstreamFailureReason) {
	p.readCallbacks.Connection().Close(types.NoFlush, types.LocalClose)
}

func (p *proxy) onUpstreamData(buffer types.IoBuffer) {
	log.DefaultLogger.Tracef("Tcp Proxy :: read upstream data , len = %v", buffer.Len())
	bytesSent := p.requestInfo.BytesSent() + uint64(buffer.Len())
	p.requestInfo.SetBytesSent(bytesSent)

	p.readCallbacks.Connection().Write(buffer.Clone())
	buffer.Drain(buffer.Len())
}

func (p *proxy) onUpstreamEvent(event types.ConnectionEvent) {
	switch event {
	case types.RemoteClose:
		p.finalizeUpstreamConnectionStats()
		p.readCallbacks.Connection().Close(types.FlushWrite, types.LocalClose)

	case types.LocalClose:
		p.finalizeUpstreamConnectionStats()
	case types.OnConnect:
	case types.Connected:
		p.readCallbacks.Connection().SetReadDisable(false)

		p.onConnectionSuccess()
	case types.ConnectTimeout:
		p.finalizeUpstreamConnectionStats()

		p.requestInfo.SetResponseFlag(types.UpstreamConnectionFailure)
		p.closeUpstreamConnection()
		p.initializeUpstreamConnection()
	case types.ConnectFailed:
		p.requestInfo.SetResponseFlag(types.UpstreamConnectionFailure)
	}
}

func (p *proxy) finalizeUpstreamConnectionStats() {
	upstreamClusterInfo := p.readCallbacks.UpstreamHost().ClusterInfo()
	upstreamClusterInfo.ResourceManager().Connections().Decrease()
}

func (p *proxy) onConnectionSuccess() {
	log.DefaultLogger.Debugf("new upstream connection %d created", p.upstreamConnection.ID())
}

func (p *proxy) onDownstreamEvent(event types.ConnectionEvent) {
	if p.upstreamConnection != nil {
		if event == types.RemoteClose {
			p.upstreamConnection.Close(types.FlushWrite, types.LocalClose)
		} else if event == types.LocalClose {
			p.upstreamConnection.Close(types.NoFlush, types.LocalClose)
		}
	}

	if event.IsClose() {
		for _, al := range p.accessLogs {
			al.Log(nil, nil, p.requestInfo)
		}
	}
}

func (p *proxy) ReadDisableUpstream(disable bool) {
	// TODO
}

func (p *proxy) ReadDisableDownstream(disable bool) {
	// TODO
}

type proxyConfig struct {
	statPrefix         string
	cluster            string
	idleTimeout        *time.Duration
	maxConnectAttempts uint32
	routes             []*route
}

type IpRangeList struct {
	cidrRanges []v2.CidrRange
}

func (ipList *IpRangeList) Contains(address net.Addr) bool {
	tcpAddr, ok := address.(*net.TCPAddr)
	log.DefaultLogger.Tracef("IpRangeList check ip = %v,address = %v", tcpAddr, address)
	if ok {
		ip := tcpAddr.IP
		for _, cidrRange := range ipList.cidrRanges {
			log.DefaultLogger.Tracef("check CidrRange = %v,ip = %v", cidrRange, ip)
			if cidrRange.IsInRange(ip) {
				return true
			}
		}
	}
	return false
}

type PortRangeList struct {
	portList []PortRange
}

func (pr *PortRangeList) Contains(address net.Addr) bool {
	tcpAddr, ok := address.(*net.TCPAddr)
	if ok {
		port := tcpAddr.Port
		log.DefaultLogger.Tracef("PortRangeList check port = %v , address = %v", port, address)
		for _, portRange := range pr.portList {
			log.DefaultLogger.Tracef("check port range , port range = %v , port = %v", portRange, port)
			if port >= portRange.min && port <= portRange.max {
				return true
			}
		}
	}
	return false
}

type PortRange struct {
	min int
	max int
}

func ParsePortRangeList(ports string) PortRangeList {
	var portList []PortRange
	if ports == "" {
		return PortRangeList{portList}
	}
	for _, portItem := range strings.Split(ports, ",") {
		if strings.Contains(portItem, "-") {
			pieces := strings.Split(portItem, "-")
			min, err := strconv.Atoi(pieces[0])
			max, err := strconv.Atoi(pieces[1])
			if err != nil {
				log.DefaultLogger.Errorf("parse port range list fail, invalid port %v", portItem)
				continue
			}
			pRange := PortRange{min: min, max: max}
			portList = append(portList, pRange)
		} else {
			port, err := strconv.Atoi(portItem)
			if err != nil {
				log.DefaultLogger.Errorf("parse port range list fail, invalid port %v", portItem)
				continue
			}
			pRange := PortRange{min: port, max: port}
			portList = append(portList, pRange)
		}
	}
	return PortRangeList{portList}
}

type route struct {
	clusterName      string
	sourceAddrs      IpRangeList
	destinationAddrs IpRangeList
	sourcePort       PortRangeList
	destinationPort  PortRangeList
}

func NewProxyConfig(config *v2.TCPProxy) ProxyConfig {
	var routes []*route

	log.DefaultLogger.Tracef("Tcp Proxy :: New Proxy Config = %v", config)
	for _, routeConfig := range config.Routes {
		route := &route{
			clusterName:      routeConfig.Cluster,
			sourceAddrs:      IpRangeList{routeConfig.SourceAddrs},
			destinationAddrs: IpRangeList{routeConfig.DestinationAddrs},
			sourcePort:       ParsePortRangeList(routeConfig.SourcePort),
			destinationPort:  ParsePortRangeList(routeConfig.DestinationPort),
		}
		log.DefaultLogger.Tracef("Tcp Proxy add one route : %v", route)

		routes = append(routes, route)
	}

	return &proxyConfig{
		statPrefix:         config.StatPrefix,
		cluster:            config.Cluster,
		idleTimeout:        config.IdleTimeout,
		maxConnectAttempts: config.MaxConnectAttempts,
		routes:             routes,
	}
}

func (pc *proxyConfig) GetRouteFromEntries(connection types.Connection) string {
	if pc.cluster != "" {
		log.DefaultLogger.Tracef("Tcp Proxy get cluster from config , cluster name = %v", pc.cluster)
		return pc.cluster
	}

	log.DefaultLogger.Tracef("Tcp Proxy get route from entries , connection = %v", connection)
	for _, r := range pc.routes {
		log.DefaultLogger.Tracef("Tcp Proxy check one route = %v", r)
		if !r.sourceAddrs.Contains(connection.RemoteAddr()) {
			continue
		}
		if !r.sourcePort.Contains(connection.RemoteAddr()) {
			continue
		}
		if !r.destinationAddrs.Contains(connection.LocalAddr()) {
			continue
		}
		if !r.destinationPort.Contains(connection.LocalAddr()) {
			continue
		}
		return r.clusterName
	}
	log.DefaultLogger.Warnf("Tcp Proxy find no cluster , connection = %v", connection)

	return ""
}

// ConnectionEventListener
// ReadFilter
type upstreamCallbacks struct {
	proxy *proxy
}

func (uc *upstreamCallbacks) OnEvent(event types.ConnectionEvent) {
	switch event {
	case types.Connected:
		uc.proxy.upstreamConnection.SetNoDelay(true)
		uc.proxy.upstreamConnection.SetReadDisable(false)
	}

	uc.proxy.onUpstreamEvent(event)
}

func (uc *upstreamCallbacks) OnData(buffer types.IoBuffer) types.FilterStatus {
	uc.proxy.onUpstreamData(buffer)
	return types.Stop
}

func (uc *upstreamCallbacks) OnNewConnection() types.FilterStatus {
	return types.Continue
}

func (uc *upstreamCallbacks) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {}

// ConnectionEventListener
type downstreamCallbacks struct {
	proxy *proxy
}

func (dc *downstreamCallbacks) OnEvent(event types.ConnectionEvent) {
	dc.proxy.onDownstreamEvent(event)
}

// LbContext is a types.LoadBalancerContext implementation
type LbContext struct {
	conn types.ReadFilterCallbacks
}

func (c *LbContext) ComputeHashKey() types.HashedValue {
	return ""
}

func (c *LbContext) MetadataMatchCriteria() types.MetadataMatchCriteria {
	return nil
}

func (c *LbContext) DownstreamConnection() net.Conn {
	return c.conn.Connection().RawConn()
}

// TCP Proxy have no header
func (c *LbContext) DownstreamHeaders() types.HeaderMap {
	return nil
}
