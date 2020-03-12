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

package xprotocol

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	str "mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

// StreamReceiver to receive keep alive response
type xprotocolKeepAlive struct {
	Codec     str.Client
	Protocol  xprotocol.XProtocol
	Timeout   time.Duration
	Threshold uint32
	Callbacks []types.KeepAliveCallback
	// runtime
	timeoutCount uint32
	idleFree     *idleFree
	// stop channel will stop all keep alive action
	once sync.Once
	stop chan struct{}
	// requests records all running request
	// a request is handled once: response or timeout
	requests map[uint64]*keepAliveTimeout
	mutex    sync.Mutex
}

func NewKeepAlive(codec str.Client, proto types.ProtocolName, timeout time.Duration, thres uint32) types.KeepAlive {
	kp := &xprotocolKeepAlive{
		Codec:        codec,
		Protocol:     xprotocol.GetProtocol(proto),
		Timeout:      timeout,
		Threshold:    thres,
		Callbacks:    []types.KeepAliveCallback{},
		timeoutCount: 0,
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
func (kp *xprotocolKeepAlive) OnEvent(event api.ConnectionEvent) {
	if event.IsClose() || event.ConnectFailure() {
		kp.Stop()
	}
}

func (kp *xprotocolKeepAlive) AddCallback(cb types.KeepAliveCallback) {
	kp.Callbacks = append(kp.Callbacks, cb)
}

func (kp *xprotocolKeepAlive) runCallback(status types.KeepAliveStatus) {
	for _, cb := range kp.Callbacks {
		cb(status)
	}
}

// SendKeepAlive will make a request to server via codec.
// use channel, do not block
func (kp *xprotocolKeepAlive) SendKeepAlive() {
	select {
	case <-kp.stop:
		return
	default:
		kp.sendKeepAlive()
	}
}

func (kp *xprotocolKeepAlive) StartIdleTimeout() {
	kp.idleFree = newIdleFree()
}

// The function will be called when connection in the codec is idle
func (kp *xprotocolKeepAlive) sendKeepAlive() {
	ctx := context.Background()
	sender := kp.Codec.NewStream(ctx, kp)
	id := sender.GetStream().ID()
	// check idle free
	if kp.idleFree.CheckFree(id) {
		kp.Codec.Close()
		return
	}
	// we send sofa rpc cmd as "header", but it maybe contains "body"
	hb := kp.Protocol.Trigger(id)
	sender.AppendHeaders(ctx, hb.GetHeader(), true)
	// start a timer for request
	kp.mutex.Lock()
	kp.requests[id] = startTimeout(id, kp)
	kp.mutex.Unlock()
}

func (kp *xprotocolKeepAlive) GetTimeout() time.Duration {
	return kp.Timeout
}

func (kp *xprotocolKeepAlive) HandleTimeout(id uint64) {
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

func (kp *xprotocolKeepAlive) HandleSuccess(id uint64) {
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

func (kp *xprotocolKeepAlive) Stop() {
	kp.once.Do(func() {
		log.DefaultLogger.Infof("[stream] [sofarpc] [keepalive] connection %d stopped keepalive", kp.Codec.ConnID())
		close(kp.stop)
	})
}

// StreamReceiver Implementation
// we just needs to make sure we can receive a response, do not care the data we received
func (kp *xprotocolKeepAlive) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	if ack, ok := headers.(xprotocol.XFrame); ok {
		kp.HandleSuccess(ack.GetRequestId())
	}
}

func (kp *xprotocolKeepAlive) OnDecodeError(ctx context.Context, err error, headers types.HeaderMap) {
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
