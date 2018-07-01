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
	"context"
	"net"
	"runtime/debug"
	"time"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/tls"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// listener impl based on golang net package
type listener struct {
	name                                  string
	localAddress                          net.Addr
	bindToPort                            bool
	listenerTag                           uint64
	perConnBufferLimitBytes               uint32
	handOffRestoredDestinationConnections bool
	cb                                    types.ListenerEventListener
	rawl                                  *net.TCPListener
	logger                                log.Logger
	tlsMng                                types.TLSContextManager
}

func NewListener(lc *v2.ListenerConfig, logger log.Logger) types.Listener {

	l := &listener{
		name:                                  lc.Name,
		localAddress:                          lc.Addr,
		bindToPort:                            lc.BindToPort,
		listenerTag:                           lc.ListenerTag,
		perConnBufferLimitBytes:               lc.PerConnBufferLimitBytes,
		handOffRestoredDestinationConnections: lc.HandOffRestoredDestinationConnections,
		logger: logger,
	}

	if lc.InheritListener != nil {
		//inherit old process's listener
		l.rawl = lc.InheritListener
	}

	l.tlsMng = tls.NewTLSServerContextManager(lc.FilterChains, l, logger)

	return l
}

func (l *listener) Name() string {
	return l.name
}

func (l *listener) Addr() net.Addr {
	return l.localAddress
}

func (l *listener) Start(lctx context.Context) {

	if l.bindToPort {
		//call listen if not inherit
		if l.rawl == nil {
			if err := l.listen(lctx); err != nil {
				// TODO: notify listener callbacks
				log.StartLogger.Fatalln(l.name, " listen failed, ", err)
				return
			}
		}

		for {
			if err := l.accept(lctx); err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					l.logger.Infof("listener %s stop accepting connections by deadline", l.name)
					return
				} else if ope, ok := err.(*net.OpError); ok {
					if !(ope.Timeout() && ope.Temporary()) {
						l.logger.Errorf("not temp-timeout error:%s", err.Error())
					}
				} else {
					l.logger.Errorf("unknown error while listener accepting:%s", err.Error())
				}
			}
		}
	}
}

func (l *listener) Stop() {
	l.rawl.SetDeadline(time.Now())
}

func (l *listener) ListenerTag() uint64 {
	return l.listenerTag
}

func (l *listener) ListenerFD() (uintptr, error) {
	file, err := l.rawl.File()
	if err != nil {
		l.logger.Errorf(" listener %s fd not found : %v", l.name, err)
		return 0, err
	}
	defer file.Close()
	return file.Fd(), nil
}

func (l *listener) PerConnBufferLimitBytes() uint32 {
	return l.perConnBufferLimitBytes
}

func (l *listener) SetListenerCallbacks(cb types.ListenerEventListener) {
	l.cb = cb
}

func (l *listener) Close(lctx context.Context) error {
	l.cb.OnClose()
	return l.rawl.Close()
}

func (l *listener) listen(lctx context.Context) error {
	var err error

	var rawl *net.TCPListener
	if rawl, err = net.ListenTCP("tcp", l.localAddress.(*net.TCPAddr)); err != nil {
		return err
	}

	l.rawl = rawl

	return nil
}

func (l *listener) accept(lctx context.Context) error {
	rawc, err := l.rawl.Accept()

	if err != nil {
		return err
	}

	// TODO: use thread pool
	go func() {
		defer func() {
			if p := recover(); p != nil {
				l.logger.Errorf("panic %v", p)

				debug.PrintStack()
			}
		}()

		if l.tlsMng != nil && l.tlsMng.Enabled() {
			rawc = l.tlsMng.Conn(rawc)
		}

		l.cb.OnAccept(rawc, l.handOffRestoredDestinationConnections)
	}()

	return nil
}
