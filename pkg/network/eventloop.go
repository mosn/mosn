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
	"sync"
	"sync/atomic"

	"mosn.io/mosn/pkg/log"

	"github.com/mosn/easygo/netpoll"
	mosnsync "mosn.io/mosn/pkg/sync"
)

var (
	// UseNetpollMode indicates which mode should be used for connection IO processing
	UseNetpollMode = false

	// read/write goroutine pool
	readPool  = mosnsync.NewWorkerPool(runtime.NumCPU())
	writePool = mosnsync.NewWorkerPool(runtime.NumCPU())

	rrCounter     uint32
	poolSize      uint32 = 1 //uint32(runtime.NumCPU())
	eventLoopPool        = make([]*eventLoop, poolSize)
	// errEventAlreadyRegistered        = errors.New("event already registered")
)

func init() {
	for i := range eventLoopPool {
		poller, err := netpoll.New(&netpoll.Config{
			OnWaitError: func(err error){
				log.DefaultLogger.Errorf("ON_WAIT error %v", err)
			},
		})
		if err != nil {
			log.DefaultLogger.Fatalf("create poller failed, caused by %v", err)
		}

		eventLoopPool[i] = &eventLoop{
			poller: poller,
			conn:   make(map[*connection]*connEvent), //TODO init size
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
	mu sync.Mutex

	poller netpoll.Poller

	conn map[*connection]*connEvent
}

func (el *eventLoop) register(conn *connection, handler *connEventHandler) error {
	// handle read
	read, err := netpoll.HandleFile(conn.rawConnection, conn.file, netpoll.EventRead|netpoll.EventOneShot)
	if err != nil {
		return err
	}

	// handle write
	write, err := netpoll.HandleFile(conn.rawConnection, conn.file, netpoll.EventWrite|netpoll.EventOneShot)
	if err != nil {
		return err
	}

	// register with wrapper
	el.poller.Start(read, el.readWrapper(conn, read, handler))
	el.poller.Start(write, el.writeWrapper(write, handler))

	el.mu.Lock()
	//store
	el.conn[conn] = &connEvent{
		read:  read,
		write: write,
	}
	el.mu.Unlock()
	return nil
}

func (el *eventLoop) registerRead(conn *connection, handler *connEventHandler) error {
	runtime.SetFinalizer(conn.file, func(f *os.File) {
		log.DefaultLogger.Errorf("GC happen on connection file %+v, conn: %+v\n", f, conn)
	})
	// handle read
	read, err := netpoll.HandleFile(conn.rawConnection, conn.file, netpoll.EventRead|netpoll.EventOneShot)
	if err != nil {
		return err
	}

	log.DefaultLogger.Errorf("register read for conn %+v", conn.RemoteAddr())
	// register

	// 为什么关闭的连接没有把注册进 ep 的 fd 清除掉
	// el.poller.Stop(read)
	el.mu.Lock()

	// el.poller.Stop(read)
	err = el.poller.Start(read, el.readWrapper(conn, read, handler))
	if err != nil {
		log.DefaultLogger.Errorf("failed to start poller %v, conn : %+v", err, conn.RemoteAddr())
	} else {
		log.DefaultLogger.Errorf("success register read for conn %+v", conn.RemoteAddr())
	}

	//store
	el.conn[conn] = &connEvent{
		read: read,
	}
	el.mu.Unlock()
	return nil
}

func (el *eventLoop) registerWrite(conn *connection, handler *connEventHandler) error {
	// handle write
	write, err := netpoll.HandleFile(conn.rawConnection, conn.file, netpoll.EventWrite|netpoll.EventOneShot)
	if err != nil {
		return err
	}

	// register
	el.poller.Start(write, el.writeWrapper(write, handler))

	el.mu.Lock()
	//store
	el.conn[conn] = &connEvent{
		write: write,
	}
	el.mu.Unlock()
	return nil
}

func (el *eventLoop) unregister(id *connection) {

	el.mu.Lock()
	defer el.mu.Unlock()
	if event, ok := el.conn[id]; ok {
		if event.read != nil {
			el.poller.Stop(event.read)
		}

		if event.write != nil {
			el.poller.Stop(event.write)
		}

		delete(el.conn, id)
	}

}

func (el *eventLoop) unregisterRead(id *connection) {
	el.mu.Lock()
	defer el.mu.Unlock()
	log.DefaultLogger.Errorf("unregister read for conn : %+v", id.RemoteAddr())
	if event, ok := el.conn[id]; ok {
		if event.read != nil {
			err := el.poller.Stop(event.read)
			if err != nil {
				log.DefaultLogger.Errorf("fail to unregister read for conn : %+v, err : %v, desc : %+v", id.RemoteAddr(), err, event.read)
			} else {
				log.DefaultLogger.Errorf("success to unregister read for conn : %+v", id.RemoteAddr())
			}
		}

		delete(el.conn, id)
	} else {
		log.DefaultLogger.Errorf("fail to unregister read for conn : %+v", id.RemoteAddr())
	}
}

func (el *eventLoop) unregisterWrite(id *connection) {
	el.mu.Lock()
	defer el.mu.Unlock()
	if event, ok := el.conn[id]; ok {
		if event.write != nil {
			el.poller.Stop(event.write)
		}

		delete(el.conn, id)
	}
}

func (el *eventLoop) readWrapper(conn *connection, desc *netpoll.Desc, handler *connEventHandler) func(netpoll.Event) {
	return func(e netpoll.Event) {
		log.DefaultLogger.Errorf("readWrapper event %v, fd : %v, desc : %+v, conn : %+v", e, desc.Fd(), desc, desc.C.RemoteAddr())
		// No more calls will be made for conn until we call epoll.Resume().
		if e&netpoll.EventReadHup != 0 {
			if !handler.onHup() {
				return
			}
		}
		readPool.Schedule(func() {
			if !handler.onRead() {
				return
			}
			err := el.poller.Resume(desc)
			if err != nil {
				log.DefaultLogger.Errorf("readWrapper RESUME err : %v, desc : %+v, conn : %+v", err, desc, desc.C.RemoteAddr())
				handler.onHup()
			}
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
