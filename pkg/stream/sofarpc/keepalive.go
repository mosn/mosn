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

package sofarpc

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	_ "github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc/codec"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/utils"
)

// StreamReceiver to receive keep alive response
type sofaRPCKeepAlive struct {
	Codec        str.Client
	ProtocolByte byte
	Timeout      time.Duration
	Threshold    uint32
	Callbacks    []types.KeepAliveCallback
	// runtime
	timeoutCount uint32
	try          chan bool
	// stop channel will stop all keep alive action
	stop chan struct{}
	// requests records all running request
	// a request is handled once: response or timeout
	requests map[uint64]*keepAliveTimeout
	mutex    sync.Mutex
}

func NewSofaRPCKeepAlive(codec str.Client, proto byte, timeout time.Duration, thres uint32) types.KeepAlive {
	kp := &sofaRPCKeepAlive{
		Codec:        codec,
		ProtocolByte: proto,
		Timeout:      timeout,
		Threshold:    thres,
		Callbacks:    []types.KeepAliveCallback{},
		timeoutCount: 0,
		try:          make(chan bool, thres),
		stop:         make(chan struct{}),
		requests:     make(map[uint64]*keepAliveTimeout),
		mutex:        sync.Mutex{},
	}
	// register keepalive to connection event listener
	// if connection is closed, keepalive should stop
	kp.Codec.AddConnectionEventListener(kp)
	return kp
}

// keepalive should stop when connection closed
func (kp *sofaRPCKeepAlive) OnEvent(event types.ConnectionEvent) {
	if event.IsClose() {
		close(kp.stop)
	}
}

func (kp *sofaRPCKeepAlive) Start() {
	for {
		select {
		case <-kp.stop:
			return
		case <-kp.try:
			kp.sendKeepAlive()
			// TODO: other action
		}
	}
}

func (kp *sofaRPCKeepAlive) AddCallback(cb types.KeepAliveCallback) {
	kp.Callbacks = append(kp.Callbacks, cb)
}

func (kp *sofaRPCKeepAlive) runCallback(status types.KeepAliveStatus) {
	for _, cb := range kp.Callbacks {
		cb(status)
	}
}

// SendKeepAlive will make a request to server via codec.
// use channel, do not block
func (kp *sofaRPCKeepAlive) SendKeepAlive() {
	select {
	case <-kp.stop:
		return
	case kp.try <- true:
	default:
		log.DefaultLogger.Warnf("keep alive too much")
	}
}

// The function will be called when connection in the codec is idle
func (kp *sofaRPCKeepAlive) sendKeepAlive() {
	ctx := context.Background()
	sender := kp.Codec.NewStream(ctx, kp)
	id := sender.GetStream().ID()
	// we send sofa rpc cmd as "header", but it maybe contains "body"
	hb := sofarpc.NewHeartbeat(kp.ProtocolByte)
	sender.AppendHeaders(ctx, hb, true)
	// start a timer for request
	kp.mutex.Lock()
	kp.requests[id] = startTimeout(id, kp)
	kp.mutex.Unlock()
}

func (kp *sofaRPCKeepAlive) GetTimeout() time.Duration {
	return kp.Timeout
}

func (kp *sofaRPCKeepAlive) HandleTimeout(id uint64) {
	select {
	case <-kp.stop:
		return
	default:
		kp.mutex.Lock()
		defer kp.mutex.Unlock()
		if _, ok := kp.requests[id]; ok {
			delete(kp.requests, id)
			atomic.AddUint32(&kp.timeoutCount, 1)
			// close the connection, stop keep alive
			if kp.timeoutCount >= kp.Threshold {
				kp.Codec.Close()
			}
			kp.runCallback(types.KeepAliveTimeout)
		}
	}
}

func (kp *sofaRPCKeepAlive) HandleSuccess(id uint64) {
	select {
	case <-kp.stop:
		return
	default:
		kp.mutex.Lock()
		defer kp.mutex.Unlock()
		if timeout, ok := kp.requests[id]; ok {
			delete(kp.requests, id)
			timeout.timer.Stop()
			// reset the tiemout count
			atomic.StoreUint32(&kp.timeoutCount, 0)
			kp.runCallback(types.KeepAliveSuccess)
		}
	}
}

func (kp *sofaRPCKeepAlive) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	kp.OnReceiveHeaders(ctx, headers, true)
}

// StreamReceiver Implementation
// we just needs to make sure we can receive a response, do not care the data we received
func (kp *sofaRPCKeepAlive) OnReceiveHeaders(ctx context.Context, headers types.HeaderMap, endStream bool) {
	if ack, ok := headers.(sofarpc.SofaRpcCmd); ok {
		kp.HandleSuccess(ack.RequestID())
	}
}

func (kp *sofaRPCKeepAlive) OnReceiveData(ctx context.Context, data types.IoBuffer, endStream bool) {
	// ignore
	// TODO: release the iobuffer
}

func (kp *sofaRPCKeepAlive) OnReceiveTrailers(ctx context.Context, trailers types.HeaderMap) {
}

func (kp *sofaRPCKeepAlive) OnDecodeError(ctx context.Context, err error, headers types.HeaderMap) {
}

//
type keepAliveTimeout struct {
	ID        uint64
	timer     *utils.Timer
	KeepAlive types.KeepAlive
}

func startTimeout(id uint64, keep types.KeepAlive) *keepAliveTimeout {
	t := &keepAliveTimeout{
		ID:        id,
		KeepAlive: keep,
	}
	t.timer = utils.NewTimer(keep.GetTimeout(), t.onTimeout)
	return t
}

func (t *keepAliveTimeout) onTimeout() {
	t.KeepAlive.HandleTimeout(t.ID)
}
