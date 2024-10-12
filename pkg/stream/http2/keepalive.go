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

package http2

import (
	"context"
	"mosn.io/pkg/variable"
	"sync"
	"time"

	atomicex "go.uber.org/atomic"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	str "mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

// StreamReceiver to receive keep alive response
type http2KeepAlive struct {
	Codec     str.Client
	Timeout   time.Duration
	Callbacks []types.KeepAliveCallback

	heartbeatFailCount atomicex.Uint32 // the number of consecutive heartbeat failures, will be reset after hb succ
	previousIsSucc     atomicex.Bool   // the previous heartbeat result
	tickCount          atomicex.Uint32 // tick intervals after the last heartbeat request sent

	idleFree *idleFree

	// once protects stop channel
	once sync.Once
	// stop channel will stop all keep alive action
	stop chan struct{}

	// mutex protects the request map
	mutex sync.Mutex
	// requests records all running request
	// a request is handled once: response or timeout
	requests map[uint64]*keepAliveTimeout
}

func (kp *http2KeepAlive) store(key uint64, val *keepAliveTimeout) {
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	kp.requests[key] = val
}

func (kp *http2KeepAlive) loadAndDelete(key uint64) (val *keepAliveTimeout, loaded bool) {
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	v, ok := kp.requests[key]
	if ok {
		delete(kp.requests, key)
	}

	return v, ok
}

// NewKeepAlive creates a keepalive object
func NewKeepAlive(codec str.Client, proto types.ProtocolName, timeout time.Duration) types.KeepAlive {
	kp := &http2KeepAlive{
		Codec:     codec,
		Timeout:   timeout,
		Callbacks: make([]types.KeepAliveCallback, 0),
		stop:      make(chan struct{}),
		requests:  make(map[uint64]*keepAliveTimeout),
	}

	// initially set previous heartbeat request success
	kp.previousIsSucc.Store(true)

	// register keepalive to connection event listener
	// if connection is closed, keepalive should stop
	kp.Codec.AddConnectionEventListener(kp)
	return kp
}

// keepalive should stop when connection closed
func (kp *http2KeepAlive) OnEvent(event api.ConnectionEvent) {
	if event.IsClose() || event.ConnectFailure() {
		kp.Stop()
	}
}

// AddCallback add a callback to keepalive
// currently there no use for this function
func (kp *http2KeepAlive) AddCallback(cb types.KeepAliveCallback) {
	kp.Callbacks = append(kp.Callbacks, cb)
}

func (kp *http2KeepAlive) runCallback(status types.KeepAliveStatus) {
	for _, cb := range kp.Callbacks {
		cb(status)
	}
}

// SendKeepAlive will make a request to server via codec.
// use channel, do not block
func (kp *http2KeepAlive) SendKeepAlive() {
	select {
	case <-kp.stop:
		return
	default:
	}

	var (
		c         = http2KeepaliveConfig.Load().(KeepaliveConfig)
		tickCount = kp.tickCount.Inc()
	)

	if kp.previousIsSucc.Load() {
		// previous hb is success
		if tickCount >= c.TickCountIfSucc {
			kp.sendKeepAlive()
		}
	} else {
		// previous hb is failure
		if tickCount >= c.TickCountIfFail {
			kp.sendKeepAlive()
		}
	}
}

func (kp *http2KeepAlive) StartIdleTimeout() {
	kp.idleFree = newIdleFree()
}

// The function will be called when connection in the codec is idle
func (kp *http2KeepAlive) sendKeepAlive() {
	ctx := context.Background()
	senderInterface := kp.Codec.NewStream(ctx, kp)
	sender, ok := senderInterface.(*clientStream)
	if !ok {
		log.DefaultLogger.Debugf("[stream] [http2] [keepalive] connection %d get bad client stream type, stream = %v", kp.Codec.ConnID(), senderInterface)
		return
	}
	id := sender.GetStream().ID()

	// check idle free
	if kp.idleFree.CheckFree(id) {
		kp.Codec.Close()
		return
	}

	// reset the tick count
	kp.tickCount.Store(0)

	kp.store(id, startTimeout(id, kp)) // store request before send, in case receive response too quick but not data in store
	// async send keepalive
	utils.GoWithRecover(func() {
		sender.sc.pingMutex.Lock()
		defer sender.sc.pingMutex.Unlock()
		sender.AppendPing(ctx)
	}, func(r interface{}) {
		log.Proxy.Errorf(ctx, "[http2 async keepalive] recover error: %v", r)
	})

	// start a timer for request
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[stream] [http2] [keepalive] connection %d send a keepalive request, id = %d", kp.Codec.ConnID(), id)
	}
}

func (kp *http2KeepAlive) GetTimeout() time.Duration {
	return kp.Timeout
}

func (kp *http2KeepAlive) HandleTimeout(id uint64) {
	c := http2KeepaliveConfig.Load().(KeepaliveConfig)
	select {
	case <-kp.stop:
		return
	default:
	}

	if _, ok := kp.loadAndDelete(id); !ok {
		return
	}

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[stream] [http2] [keepalive] connection %d receive a request timeout %d", kp.Codec.ConnID(), id)
	}

	kp.heartbeatFailCount.Inc()
	kp.previousIsSucc.Store(false)

	// close the connection, stop keep alive
	if kp.heartbeatFailCount.Load() >= c.FailCountToClose {
		kp.Codec.Close()
	}
	kp.runCallback(types.KeepAliveTimeout)
}

func (kp *http2KeepAlive) HandleSuccess(id uint64) {
	select {
	case <-kp.stop:
		return
	default:
	}

	timeout, ok := kp.loadAndDelete(id)
	if !ok {
		return
	}

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[stream] [http2] [keepalive] connection %d receive a response success %d", kp.Codec.ConnID(), id)
	}

	timeout.timer.Stop()

	// reset the tiemout count
	kp.heartbeatFailCount.Store(0)
	kp.previousIsSucc.Store(true)

	kp.runCallback(types.KeepAliveSuccess)
}

func (kp *http2KeepAlive) Stop() {
	kp.once.Do(func() {
		log.DefaultLogger.Infof("[stream] [http2] [keepalive] connection %d stopped keepalive", kp.Codec.ConnID())
		close(kp.stop)
	})
}

// OnReceive StreamReceiver Implementation
// we just needs to make sure we can receive a response, do not care the data we received
func (kp *http2KeepAlive) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	streamID, _ := variable.Get(ctx, types.VariableStreamID)
	kp.HandleSuccess(uint64(streamID.(uint32)))

}

// OnDecodeError does not process decode failure
// the timer will fail this heart beat
func (kp *http2KeepAlive) OnDecodeError(ctx context.Context, err error, headers types.HeaderMap) {

}

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
