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
	"errors"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/neverhook/easygo/netpoll"
	mosnsync "mosn.io/mosn/pkg/sync"
)

var (
	// UseNetpollMode indicates which mode should be used for connection IO processing
	UseNetpollMode = false

	// read/write goroutine pool
	readPool  = mosnsync.NewWorkerPool(runtime.NumCPU())
	writePool = mosnsync.NewWorkerPool(runtime.NumCPU())

	rrCounter                 uint32
	poolSize                  uint32 = 1 //uint32(runtime.NumCPU())
	eventLoopPool                    = make([]*eventLoop, poolSize)
	errEventAlreadyRegistered        = errors.New("event already registered")
)

func init() {
	//for i := range eventLoopPool {
	//	poller, err := netpoll.New(nil)
	//	if err != nil {
	//		log.Fatalln("create poller failed, caused by ", err)
	//	}
	//
	//	eventLoopPool[i] = &eventLoop{
	//		poller: poller,
	//		conn:   make(map[uint64]*connEvent), //TODO init size
	//	}
	//}
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
	mu sync.Mutex

	poller netpoll.Poller

	conn map[uint64]*connEvent
}

func (el *eventLoop) register(conn *connection, handler *connEventHandler) error {
	// handle read
	read, err := netpoll.HandleFile(conn.file, netpoll.EventRead|netpoll.EventOneShot)
	if err != nil {
		return err
	}

	// handle write
	write, err := netpoll.HandleFile(conn.file, netpoll.EventWrite|netpoll.EventOneShot)
	if err != nil {
		return err
	}

	// register with wrapper
	el.poller.Start(read, el.readWrapper(read, handler))
	el.poller.Start(write, el.writeWrapper(write, handler))

	el.mu.Lock()
	//store
	el.conn[conn.id] = &connEvent{
		read:  read,
		write: write,
	}
	el.mu.Unlock()
	return nil
}

func (el *eventLoop) registerRead(conn *connection, handler *connEventHandler) error {
	// handle read
	read, err := netpoll.HandleFile(conn.file, netpoll.EventRead|netpoll.EventOneShot)
	if err != nil {
		return err
	}

	// register
	el.poller.Start(read, el.readWrapper(read, handler))

	el.mu.Lock()
	//store
	el.conn[conn.id] = &connEvent{
		read: read,
	}
	el.mu.Unlock()
	return nil
}

func (el *eventLoop) registerWrite(conn *connection, handler *connEventHandler) error {
	// handle write
	write, err := netpoll.HandleFile(conn.file, netpoll.EventWrite|netpoll.EventOneShot)
	if err != nil {
		return err
	}

	// register
	el.poller.Start(write, el.writeWrapper(write, handler))

	el.mu.Lock()
	//store
	el.conn[conn.id] = &connEvent{
		write: write,
	}
	el.mu.Unlock()
	return nil
}

func (el *eventLoop) unregister(id uint64) {

	if event, ok := el.conn[id]; ok {
		if event.read != nil {
			el.poller.Stop(event.read)
		}

		if event.write != nil {
			el.poller.Stop(event.write)
		}

		el.mu.Lock()
		delete(el.conn, id)
		el.mu.Unlock()
	}

}

func (el *eventLoop) unregisterRead(id uint64) {
	if event, ok := el.conn[id]; ok {
		if event.read != nil {
			el.poller.Stop(event.read)
		}

		el.mu.Lock()
		delete(el.conn, id)
		el.mu.Unlock()
	}
}

func (el *eventLoop) unregisterWrite(id uint64) {
	if event, ok := el.conn[id]; ok {
		if event.write != nil {
			el.poller.Stop(event.write)
		}

		el.mu.Lock()
		delete(el.conn, id)
		el.mu.Unlock()
	}
}

func (el *eventLoop) readWrapper(desc *netpoll.Desc, handler *connEventHandler) func(netpoll.Event) {
	return func(e netpoll.Event) {
		// No more calls will be made for conn until we call epoll.Resume().
		if e&netpoll.EventReadHup != 0 {
			el.poller.Stop(desc)
			if !handler.onHup() {
				return
			}
		}
		readPool.Schedule(func() {
			if !handler.onRead() {
				return
			}
			el.poller.Resume(desc)
		})
	}
}

func (el *eventLoop) writeWrapper(desc *netpoll.Desc, handler *connEventHandler) func(netpoll.Event) {
	return func(e netpoll.Event) {
		// No more calls will be made for conn until we call epoll.Resume().
		if e&netpoll.EventWriteHup != 0 {
			el.poller.Stop(desc)
			if !handler.onHup() {
				return
			}
		}
		writePool.ScheduleAlways(func() {
			if !handler.onWrite() {
				return
			}
			el.poller.Resume(desc)
		})
	}
}
