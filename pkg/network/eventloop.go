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
	"runtime"
	"sync/atomic"

	"mosn.io/mosn/pkg/log"

	"github.com/mosn/easygo/netpoll"
	mosnsync "mosn.io/mosn/pkg/sync"
)

var (
	// UseNetpollMode indicates which mode should be used for connection IO processing
	UseNetpollMode = false

	// read/write goroutine pool
	readPool  = mosnsync.NewWorkerPool(2 * runtime.NumCPU())
	writePool = mosnsync.NewWorkerPool(2 * runtime.NumCPU())

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

func attach() *eventLoop {
	return eventLoopPool[atomic.AddUint32(&rrCounter, 1)%poolSize]
}

type connEvent struct {
	read  *netpoll.Desc
	write *netpoll.Desc
}

type connEventHandler struct {
	onHup   func() bool
	onRead  func() bool
	onWrite func() bool
}

type eventLoop struct {
	poller netpoll.Poller
}

func (el *eventLoop) registerRead(conn *connection, handler *connEventHandler) error {
	// handle read
	read, err := netpoll.HandleFile(conn.file, netpoll.EventRead|netpoll.EventOneShot)
	if err != nil {
		return err
	}

	// register
	err = el.poller.Start(read, el.readWrapper(conn, read, handler))
	if err != nil {
		log.DefaultLogger.Errorf("[network] registerRead failed err : %v, addr: %v", err, conn.RemoteAddr())
		return err
	}

	//store
	conn.ev = &connEvent{
		read: read,
	}
	return nil
}

func (el *eventLoop) unregisterRead(conn *connection) {
	if conn.ev != nil {
		err := el.poller.Stop(conn.ev.read)
		if err != nil {
			log.DefaultLogger.Errorf("[unregisterRead] failed, %v", err)
		}
	}
}

func (el *eventLoop) readWrapper(conn *connection, desc *netpoll.Desc, handler *connEventHandler) func(netpoll.Event) {
	return func(e netpoll.Event) {
		// No more calls will be made for conn until we call epoll.Resume().
		readPool.Schedule(func() {

			if !handler.onRead() {
				return
			}

			err := el.poller.Resume(desc)
			if err != nil {
				log.DefaultLogger.Errorf("RESUME failed, err : %v, desc : %v, raddr : %v, laddr : %v", err, desc, conn.RemoteAddr(), conn.LocalAddr())
			}
		})
	}
}
