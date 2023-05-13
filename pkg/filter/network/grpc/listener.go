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

package grpc

import (
	"errors"
	"net"
	"syscall"
	"time"

	"go.uber.org/atomic"
	"mosn.io/mosn/pkg/log"
)

// Listener is an implementation of net.Listener
type Listener struct {
	closed  atomic.Bool
	accepts chan net.Conn // TODO: use a queue instead of channel to avoid hang up
	addr    net.Addr
}

func NewListener(address string) (*Listener, error) {
	return createListener(networkTcp, address)
}

func createListener(network, address string) (*Listener, error) {
	var (
		addr net.Addr
		err  error
	)
	if network == networkUnix {
		addr, err = net.ResolveUnixAddr(networkUnix, address)
	} else {
		addr, err = net.ResolveTCPAddr(networkTcp, address)
	}
	if err != nil {
		log.DefaultLogger.Errorf("invalid server address info: %s, error: %v", address, err)
		return nil, err
	}
	return &Listener{
		accepts: make(chan net.Conn, 10),
		addr:    addr,
	}, nil
}

func NewUnixListener(address string) (*Listener, error) {
	return createListener(networkUnix, address)
}

var _ net.Listener = (*Listener)(nil)

func (l *Listener) Accept() (net.Conn, error) {
	c, ok := <-l.accepts
	if !ok {
		return nil, syscall.EINVAL
	}
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[grpc] listener %s receive a connection", l.addr.String())
	}
	return c, nil
}

func (l *Listener) Addr() net.Addr {
	return l.addr
}

func (l *Listener) Close() error {
	if !l.closed.CAS(false, true) {
		return syscall.EINVAL
	}
	close(l.accepts)
	return nil
}

func (l *Listener) NewConnection(conn net.Conn) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("[grpc] listener has been closed, send on closed channel")
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[grpc] listener has been closed, %v", r)
			}
			conn.Close()
			err = errors.New("listener closed")
		}
	}()
	timer := time.NewTimer(3 * time.Second)
	select {
	case l.accepts <- conn:
		timer.Stop()
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[grpc] listener %s new a connection, wait to accept", l.addr.String())
		}
	case <-timer.C:
		log.DefaultLogger.Errorf("[grpc] connection buffer full, and timeout")
		conn.Close()
		err = errors.New("accept connection timeout")
	}
	return
}
