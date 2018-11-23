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
	"context"
	"net"
	"reflect"
	"sync/atomic"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// A shadow stream is a created by downstream, but will never write response
// A shadow stream is created when a upstream request is sent, so the stream filters is not in shadow
// no request info to log and no accesslog
type shadowDownstream struct {
	streamID        string
	proxy           *proxy
	route           types.Route
	cluster         types.ClusterInfo
	upstreamRequest *shadowUpstreamRequest

	// timeout makes stream release if no repsonse
	timeout  *Timeout
	reqTimer *timer

	downstreamReqHeaders  types.HeaderMap
	downstreamReqDataBuf  types.IoBuffer
	downstreamReqTrailers types.HeaderMap

	upstreamRequestSent bool
	upstreamProcessDone bool

	downstreamReset   uint32
	downstreamCleaned uint32
	upstreamReset     uint32

	context  context.Context
	logger   log.Logger
	snapshot types.ClusterSnapshot
}

func newShadowDownstream(down *downStream) *shadowDownstream {
	ctx := buffer.NewBufferPoolContext(context.Background(), true)
	shadowBuffers := shadowBuffersByContext(ctx)
	stream := &shadowBuffers.stream
	// stream id add shadow
	stream.streamID = down.streamID
	stream.proxy = down.proxy
	stream.context = ctx
	stream.logger = log.ByContext(down.proxy.context)

	// TODO: stats for shadow

	if !stream.fillWithDownstream(down) {
		return nil
	}
	return stream
}

// fillWithDownstream use downstream to fill up request info
func (s *shadowDownstream) fillWithDownstream(down *downStream) bool {
	s.route = down.route
	if s.route == nil || s.route.RouteRule().Policy() == nil || s.route.RouteRule().Policy().ShadowPolicy() == nil {
		s.logger.Errorf("shadow policy is nil")
		return false
	}
	clusterName := s.route.RouteRule().Policy().ShadowPolicy().ClusterName()
	clusterSnapshot := s.proxy.clusterManager.GetClusterSnapshot(context.Background(), clusterName)
	if reflect.ValueOf(clusterSnapshot).IsNil() {
		s.logger.Errorf("shadow cluster snapshot is nil, cluster name is: %s", clusterName)
		return false
	}
	s.snapshot = clusterSnapshot
	s.cluster = clusterSnapshot.ClusterInfo()
	s.logger.Tracef("get shadow trafic's cluster: %s", clusterName)

	currentProtocol := types.Protocol(s.proxy.config.UpstreamProtocol)
	connPool := s.proxy.clusterManager.ConnPoolForCluster(s, clusterSnapshot, currentProtocol)
	if connPool == nil {
		s.logger.Errorf("no healthy shadow cluster %s", clusterName)
		return false
	}
	s.logger.Tracef("after choose conn pool")
	s.downstreamReqHeaders = down.downstreamReqHeaders
	// needs clone
	if down.downstreamReqDataBuf != nil {
		s.downstreamReqDataBuf = down.downstreamReqDataBuf.Clone()
	}
	if down.downstreamReqTrailers != nil {
		// TODO: also do a copy
		s.downstreamReqTrailers = down.downstreamReqTrailers
	}
	s.timeout = parseProxyTimeout(s.route, s.downstreamReqHeaders)
	// Build upstream request
	shadowBuffers := shadowBuffersByContext(s.context)
	s.upstreamRequest = &shadowBuffers.request
	s.upstreamRequest.downStream = s
	s.upstreamRequest.proxy = s.proxy
	s.upstreamRequest.connPool = connPool

	return true
}
func (s *shadowDownstream) endStream() {
	s.cleanStream()
}

// Clean up on end stream or reset stream
func (s *shadowDownstream) cleanStream() {
	if !atomic.CompareAndSwapUint32(&s.downstreamCleaned, 0, 1) {
		return
	}
	if s.upstreamRequest != nil {
		s.upstreamRequest.resetStream()
	}
	s.cleanUp()
	s.GiveStream()
}

func (s *shadowDownstream) cleanUp() {
	if s.upstreamRequest != nil {
		s.upstreamRequest.requestSender = nil
	}
	if s.reqTimer != nil {
		s.reqTimer.stop()
		s.reqTimer = nil
	}
}

func (s *shadowDownstream) GiveStream() {
	s.logger.Tracef("shadow stream clean, stream id = %s", s.streamID)
	if s.snapshot != nil {
		s.proxy.clusterManager.PutClusterSnapshot(s.snapshot)
	}
	if s.upstreamReset == 1 || s.downstreamReset == 1 {
		return
	}
	if s.downstreamReqDataBuf != nil {
		buffer.PutIoBuffer(s.downstreamReqDataBuf)
	}
	if ctx := buffer.PoolContext(s.context); ctx != nil {
		ctx.Give()
	}
}

func (s *shadowDownstream) OnResetStream(reason types.StreamResetReason) {
	if !atomic.CompareAndSwapUint32(&s.downstreamReset, 0, 1) {
		return
	}
	s.cleanStream()
}

func (s *shadowDownstream) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
	// if active stream finished the lifecycle, just ignore further data
	if s.upstreamProcessDone {
		return
	}
	s.logger.Errorf("shadow stream decode error, stream id = %s, error = %v", s.streamID, err)
	s.OnResetStream(types.StreamLocalReset)
}

func (s *shadowDownstream) onUpstreamRequestSent() {
	s.upstreamRequestSent = true
	if s.upstreamRequest != nil {
		if s.timeout.GlobalTimeout > 0 {
			if s.reqTimer != nil {
				s.reqTimer.stop()
			}
			s.reqTimer = newTimer(s.onTimeout, s.timeout.GlobalTimeout)
		}
	}
}

func (s *shadowDownstream) onTimeout() {
	// TODO stats
	if s.upstreamRequest != nil {
		s.upstreamRequest.resetStream()
	}
	s.onUpstreamReset(UpstreamGlobalTimeout, types.StreamLocalReset)
}

func (s *shadowDownstream) sendRequest() {
	s.upstreamRequest.appendHeaders(s.downstreamReqHeaders, s.downstreamReqDataBuf == nil && s.downstreamReqTrailers == nil)
	if s.downstreamReqDataBuf != nil {
		s.upstreamRequest.appendData(s.downstreamReqDataBuf, s.downstreamReqTrailers == nil)
	}
	if s.downstreamReqTrailers != nil {
		s.upstreamRequest.appendTrailers(s.downstreamReqTrailers)
	}
	s.onUpstreamRequestSent()
}

func (s *shadowDownstream) onUpstreamReset(urtype UpstreamResetType, reason types.StreamResetReason) {
	if !atomic.CompareAndSwapUint32(&s.upstreamReset, 0, 1) {
		return
	}
	s.endStream()
}

// LoadBalancerContext implement
func (s *shadowDownstream) ComputeHashKey() types.HashedValue {
	return ""
}
func (s *shadowDownstream) MetadataMatchCriteria() types.MetadataMatchCriteria {
	if s.route != nil && s.route.RouteRule() != nil {
		return s.route.RouteRule().MetadataMatchCriteria(s.cluster.Name())
	}
	return nil
}
func (s *shadowDownstream) DownstreamConnection() net.Conn {
	// TODO: shadow should have no down stream connection
	return s.proxy.readCallbacks.Connection().RawConn()
}
func (s *shadowDownstream) DownstreamHeaders() types.HeaderMap {
	return s.downstreamReqHeaders
}
