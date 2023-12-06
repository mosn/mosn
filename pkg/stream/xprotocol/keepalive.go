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
	"time"

	atomicex "go.uber.org/atomic"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	str "mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

var (
	nilTask = &fastKeepAliveTask{}
)

// StreamReceiver to receive keep alive response
type xprotocolKeepAlive struct {
	Codec     str.Client
	Protocol  api.XProtocol
	Timeout   time.Duration
	Callbacks []types.KeepAliveCallback

	heartbeatFailCount atomicex.Uint32 // the number of consecutive heartbeat failures, will be reset after hb succ
	previousIsSucc     atomicex.Bool   // the previous heartbeat result
	tickCount          atomicex.Uint32 // tick intervals after the last heartbeat request sent
	fastFailTask       atomicex.Value  // fast fail task for keepalive check.

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

func (kp *xprotocolKeepAlive) store(key uint64, val *keepAliveTimeout) {
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	kp.requests[key] = val
}

func (kp *xprotocolKeepAlive) loadAndDelete(key uint64) (val *keepAliveTimeout, loaded bool) {
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	v, ok := kp.requests[key]
	if ok {
		delete(kp.requests, key)
	}

	return v, ok
}

// NewKeepAlive creates a keepalive object
func NewKeepAlive(codec str.Client, proto api.XProtocol, timeout time.Duration) types.KeepAlive {
	kp := &xprotocolKeepAlive{
		Codec:     codec,
		Protocol:  proto,
		Timeout:   timeout,
		Callbacks: make([]types.KeepAliveCallback, 0),
		stop:      make(chan struct{}),
		requests:  make(map[uint64]*keepAliveTimeout),
	}

	// initially set previous heartbeat request success
	kp.previousIsSucc.Store(true)

	// initially setup sets the heartbeat check fast failure task to nil
	kp.fastFailTask.Store(nilTask)

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

// AddCallback add a callback to keepalive
// currently there no use for this function
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
	}

	var (
		c         = xprotoKeepaliveConfig.Load().(KeepaliveConfig)
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

	// reset the tick count
	kp.tickCount.Store(0)

	// we send sofa rpc cmd as "header", but it maybe contains "body"
	hb := kp.Protocol.Trigger(ctx, id)
	kp.store(id, startTimeout(id, kp)) // store request before send, in case receive response too quick but not data in store
	sender.AppendHeaders(ctx, hb.GetHeader(), true)
	// start a timer for request
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[stream] [xprotocol] [keepalive] connection %d send a keepalive request, id = %d", kp.Codec.ConnID(), id)
	}
}

func (kp *xprotocolKeepAlive) GetTimeout() time.Duration {
	return kp.Timeout
}

func (kp *xprotocolKeepAlive) startFastKeepAliveTask() {
	select {
	case <-kp.stop:
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[stream] [xprotocol] [keepalive] connection %d allready stop keepalive, skip start fast fail task", kp.Codec.ConnID())
		}
		return
	default:
	}

	if v := kp.fastFailTask.Load(); v == nilTask {
		task := NewFastKeepAliveTask(kp.Codec.ConnID(), kp.sendKeepAlive, kp.stop)
		if kp.fastFailTask.CompareAndSwap(nilTask, task) {
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[stream] [xprotocol] [keepalive] connection %d start fast fail task", kp.Codec.ConnID())
			}
			task.Start()
		}
	}
}

func (kp *xprotocolKeepAlive) stopFastKeepAliveTask() {
	if v := kp.fastFailTask.Load(); v != nilTask {
		task := v.(*fastKeepAliveTask)
		if kp.fastFailTask.CompareAndSwap(task, nilTask) {
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[stream] [xprotocol] [keepalive] connection %d stop fast fail task", kp.Codec.ConnID())
			}
			task.Stop()
		}
	}
}

func (kp *xprotocolKeepAlive) HandleTimeout(id uint64) {
	c := xprotoKeepaliveConfig.Load().(KeepaliveConfig)
	select {
	case <-kp.stop:
		return
	default:
	}

	if _, ok := kp.loadAndDelete(id); !ok {
		return
	}

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[stream] [xprotocol] [keepalive] connection %d receive a request timeout %d", kp.Codec.ConnID(), id)
	}

	kp.heartbeatFailCount.Inc()
	kp.previousIsSucc.Store(false)

	// close the connection, stop keep alive
	if kp.heartbeatFailCount.Load() >= c.FailCountToClose {
		kp.Codec.Close()
	}
	kp.runCallback(types.KeepAliveTimeout)

	if c.FastFail {
		// start the heartbeat check fast failure task when the heartbeat check fails
		kp.startFastKeepAliveTask()
	}
}

func (kp *xprotocolKeepAlive) HandleSuccess(id uint64) {
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
		log.DefaultLogger.Debugf("[stream] [xprotocol] [keepalive] connection %d receive a request success %d", kp.Codec.ConnID(), id)
	}

	timeout.timer.Stop()

	// reset the tiemout count
	kp.heartbeatFailCount.Store(0)
	kp.previousIsSucc.Store(true)

	// stop the heartbeat check fast failure task when the heartbeat check success
	kp.stopFastKeepAliveTask()

	kp.runCallback(types.KeepAliveSuccess)
}

func (kp *xprotocolKeepAlive) Stop() {
	kp.once.Do(func() {
		log.DefaultLogger.Infof("[stream] [xprotocol] [keepalive] connection %d stopped keepalive", kp.Codec.ConnID())
		close(kp.stop)
		kp.stopFastKeepAliveTask()
	})
}

// StreamReceiver Implementation
// we just needs to make sure we can receive a response, do not care the data we received
func (kp *xprotocolKeepAlive) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	if ack, ok := headers.(api.XFrame); ok {
		kp.HandleSuccess(ack.GetRequestId())
	}
}

// OnDecodeError does not process decode failure
// the timer will fail this heart beat
func (kp *xprotocolKeepAlive) OnDecodeError(ctx context.Context, err error, headers types.HeaderMap) {

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

func NewFastKeepAliveTask(connID uint64, sendKeepAlive func(), stopKeepAlive chan struct{}) *fastKeepAliveTask {
	f := &fastKeepAliveTask{
		connID:        connID,
		sendKeepAlive: sendKeepAlive,
		stopKeepAlive: stopKeepAlive,
	}
	f.start.Store(false)
	return f
}

type fastKeepAliveTask struct {
	connID        uint64
	sendKeepAlive func()
	start         atomicex.Bool
	stopTask      chan struct{}
	stopKeepAlive chan struct{}
}

func (f *fastKeepAliveTask) Start() {
	if f.start.CAS(false, true) {
		f.stopTask = make(chan struct{})
		go f.safeFastSendKeepAliveLoop()
	}
}

func (f *fastKeepAliveTask) safeFastSendKeepAliveLoop() {
	defer func() {
		if err := recover(); err != nil {
			log.DefaultLogger.Debugf("[stream] [xprotocol] [keepalive] [fastfail] connection %d fast send keepalive loop panic: %v", f.connID, err)
		}
	}()

	var (
		c      = xprotoKeepaliveConfig.Load().(KeepaliveConfig)
		ticker = time.NewTicker(c.FastSendInterval)
	)

	for {
		select {
		case <-f.stopKeepAlive:
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[stream] [xprotocol] [keepalive] [fastfail] connection %d allready stop keepalive, quit fast fail task", f.connID)
			}
			return
		case <-f.stopTask:
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[stream] [xprotocol] [keepalive] [fastfail] connection %d already stop fast fail task", f.connID)
			}
			return
		case <-ticker.C:
			if c.FastFail {
				if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
					log.DefaultLogger.Debugf("[stream] [xprotocol] [keepalive] [fastfail] connection %d send a keepalive request", f.connID)
				}
				f.sendKeepAlive()
			} else {
				return
			}
		}
	}
}

func (f *fastKeepAliveTask) Stop() {
	if f.start.CAS(true, false) {
		close(f.stopTask)
	}
}
