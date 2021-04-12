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
	"net"
	"syscall"

	"go.uber.org/atomic"
	"mosn.io/mosn/pkg/log"
)

// Listener is an implementation of net.Listener
type Listener struct {
	closed  atomic.Bool
	accepts chan net.Conn
	addr    net.Addr
}

func NewListener(conf map[string]interface{}) *Listener {
	var (
		addr net.Addr
		err  error
	)
	// TODO: load network/address config from listener config
	v := conf["address"]
	if addrstr, ok := v.(string); ok {
		addr, err = net.ResolveTCPAddr("tcp", addrstr)
		if err != nil {
			log.DefaultLogger.Errorf("invalid server address info: %s, error: %v", addrstr, err)
		}
	}
	if addr == nil {
		log.DefaultLogger.Warnf("grpc listener: no address config found, use an empty instead")
		addr = &net.TCPAddr{} // set an empty addr
	}
	return &Listener{
		accepts: make(chan net.Conn),
		addr:    addr,
	}
}

var _ net.Listener = &Listener{}

func (l *Listener) Accept() (net.Conn, error) {
	c, ok := <-l.accepts
	if !ok {
		return nil, syscall.EINVAL
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

func (l *Listener) NewConnection(conn net.Conn) {
	l.accepts <- conn
}
