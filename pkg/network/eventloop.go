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

package network

import (
	"os"
	"runtime"
	"sync/atomic"

	atomicex "go.uber.org/atomic"
	"mosn.io/mosn/pkg/log"
	mosnsync "mosn.io/mosn/pkg/sync"
	"mosn.io/pkg/netpoll"
)

var (
	// UseNetpollMode indicates which mode should be used for connection IO processing
	UseNetpollMode = false

	// read/write goroutine pool
	readPool = mosnsync.NewWorkerPool(4 * runtime.NumCPU())

	rrCounter     uint32
	poolSize      uint32 = 1 //uint32(runtime.NumCPU())
	eventLoopPool        = make([]*eventLoop, poolSize)
)

func init() {
	for i := range eventLoopPool {
		poller, err := netpoll.New(nil)
		if err != nil {
			log.DefaultLogger.Fatalf("create poller failed, caused by %v", err)
		}

		eventLoopPool[i] = &eventLoop{
			poller: poller,
		}
	}
}

func init() {
	if os.Getenv("NETPOLL") == "on" {
		log.DefaultLogger.Infof("[network] set netpoll to true by env")
		SetNetpollMode(true)
	}
}

// SetNetpollMode set the netpoll mode
func SetNetpollMode(enable bool) {
	UseNetpollMode = enable
}

func attach() *eventLoop {
	return eventLoopPool[atomic.AddUint32(&rrCounter, 1)%poolSize]
}

type connEvent struct {
	readDesc *netpoll.Desc
	stopped  atomicex.Bool // default is 0/false
}

type connEventHandler struct {
	onHup  func() bool
	onRead func() bool
}

type eventLoop struct {
	poller netpoll.Poller
}

func (el *eventLoop) registerRead(conn *connection, handler *connEventHandler) error {
	// handle read
	desc, err := netpoll.HandleFile(conn.file, netpoll.EventRead|netpoll.EventOneShot)
	if err != nil {
		return err
	}

	// store should happen before register
	// if conn Close happen immediately after poller.Start
	// the conn.ev is nil, so this connection fd will not be removed from epfd
	conn.poll.ev = &connEvent{
		readDesc: desc,
	}

	readCallback := func(e netpoll.Event) {
		// No more calls will be made for conn until we call epoll.Resume().
		readPool.ScheduleAuto(func() {
			if !handler.onRead() {
				return
			}

			// if conn is closed by local already
			if conn.poll.ev.stopped.Load() {
				return
			}

			err := el.poller.Resume(conn.poll.ev.readDesc)
			if err != nil {
				log.DefaultLogger.Errorf("RESUME failed, err : %v, desc : %v, raddr : %v, laddr : %v", err,
					desc, conn.RemoteAddr(), conn.LocalAddr())
			}
		})
	}

	// register
	err = el.poller.Start(desc, readCallback)

	if err != nil {
		log.DefaultLogger.Errorf("[network] registerRead failed err : %v, addr: %v", err, conn.RemoteAddr())
		return err
	}

	return nil
}

func (el *eventLoop) unregisterRead(conn *connection) {
	if conn.poll.ev != nil {
		conn.poll.ev.stopped.Store(true)
		err := el.poller.Stop(conn.poll.ev.readDesc)
		if err != nil {
			log.DefaultLogger.Errorf("[unregisterRead] failed, %v", err)
		}
	}
}
