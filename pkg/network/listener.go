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
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/stagemanager"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

type ListenerState int

// listener state
// ListenerInited means listener is inited, a inited listener can be started or stopped
// ListenerRunning means listener is running, start a running listener will be ignored.
// ListenerStopped means listener is stopped.
// ListenerClosed  means listener is closed, start a closed listener without restart flag will be ignored.
const (
	ListenerInited ListenerState = iota
	ListenerRunning
	ListenerStopped
	ListenerClosed
)

// Factory function for creating mosn listener.
type ListenerFactory func(lc *v2.Listener) types.Listener

var listenerFactory ListenerFactory = NewListener

func GetListenerFactory() ListenerFactory {
	return listenerFactory
}

func RegisterListenerFactory(factory ListenerFactory) {
	if factory != nil {
		listenerFactory = factory
	}
}

// listener impl based on golang net package
type listener struct {
	name                    string
	localAddress            net.Addr
	bindToPort              bool
	listenerTag             uint64
	perConnBufferLimitBytes uint32
	OriginalDst             v2.OriginalDstType
	network                 string
	cb                      types.ListenerEventListener
	packetConn              net.PacketConn
	rawl                    net.Listener
	config                  *v2.Listener
	mutex                   sync.Mutex
	// listener state indicates the listener's running state. The listener state effects if a listener binded to a port
	state ListenerState
}

func NewListener(lc *v2.Listener) types.Listener {

	l := &listener{
		name:                    lc.Name,
		localAddress:            lc.Addr,
		bindToPort:              lc.BindToPort,
		listenerTag:             lc.ListenerTag,
		perConnBufferLimitBytes: lc.PerConnBufferLimitBytes,
		OriginalDst:             lc.OriginalDst,
		network:                 lc.Network,
		config:                  lc,
	}

	if lc.InheritListener != nil {
		//inherit old process's listener
		l.rawl = lc.InheritListener
	}

	if lc.InheritPacketConn != nil {
		l.packetConn = *lc.InheritPacketConn
	}

	if lc.Network == "" {
		l.network = "tcp"
	}
	lc.Network = strings.ToLower(lc.Network)
	return l
}

func (l *listener) IsBindToPort() bool {
	return l.bindToPort
}

func (l *listener) Config() *v2.Listener {
	return l.config
}

func (l *listener) SetConfig(config *v2.Listener) {
	l.config = config
}

func (l *listener) Name() string {
	return l.name
}

func (l *listener) Addr() net.Addr {
	return l.localAddress
}

func (l *listener) Start(lctx context.Context, restart bool) {
	if l.bindToPort {
		ignore := func() bool {
			l.mutex.Lock()
			defer l.mutex.Unlock()
			switch l.state {
			case ListenerRunning:
				// if listener is running, ignore start
				log.DefaultLogger.Debugf("[network] [listener start] %s is running", l.name)
				return true
			case ListenerStopped:
				if err := l.setDeadline(time.Time{}); err != nil {
					log.DefaultLogger.Alertf("listener.start", "[network] [listener start] [listen] %s reset deadline failed, %v", l.name, err)
				}
			case ListenerClosed:
				if !restart {
					return true
				}
				log.DefaultLogger.Infof("[network] [listener start] %s restart listener ", l.name)
				if err := l.listen(lctx); err != nil {
					// TODO: notify listener callbacks
					log.DefaultLogger.Alertf("listener.start", "[network] [listener start] [listen] %s listen failed, %v", l.name, err)
					return true
				}
			default:
				// try start listener
				//call listen if not inherit
				if l.rawl == nil && l.packetConn == nil {
					if err := l.listen(lctx); err != nil {
						// TODO: notify listener callbacks
						log.StartLogger.Fatalf("[network] [listener start] [listen] %s listen failed, %v", l.name, err)
					}
				}
			}
			l.state = ListenerRunning
			// add metrics for listener if bind port
			switch l.network {
			case "udp":
				metrics.AddListenerAddr(l.packetConn.(*net.UDPConn).LocalAddr().String() + l.network)
			default:
				metrics.AddListenerAddr(l.rawl.Addr().String())
			}

			return false
		}()
		if ignore {
			return
		}
		switch l.network {
		case "udp":
			l.readMsgEventLoop(lctx)
		default:
			l.acceptEventLoop(lctx)
		}
	}
}

func (l *listener) acceptEventLoop(lctx context.Context) {
	for {
		if err := l.accept(lctx); err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				log.DefaultLogger.Infof("[network] [listener start] [accept] listener %s stop accepting connections by deadline", l.name)
				return
			}
			if ope, ok := err.(*net.OpError); ok {
				// not timeout error and not temporary, which means the error is non-recoverable
				// stop accepting loop and log the event
				if !(ope.Timeout() && ope.Temporary()) {
					// accept error raised by sockets closing
					if ope.Op == "accept" {
						log.DefaultLogger.Infof("[network] [listener start] [accept] listener %s %s closed", l.name, l.Addr())
					} else {
						log.DefaultLogger.Alertf("listener.accept", "[network] [listener start] [accept] listener %s occurs non-recoverable error, stop listening and accepting:%s", l.name, err.Error())
					}
					return
				}
			} else {
				log.DefaultLogger.Errorf("[network] [listener start] [accept] listener %s occurs unknown error while accepting:%s", l.name, err.Error())
			}
		}
	}
}

func (l *listener) readMsgEventLoop(lctx context.Context) {
	utils.GoWithRecover(func() {
		readMsgLoop(lctx, l)
	}, func(r interface{}) {
		l.readMsgEventLoop(lctx)
	})
}

// Shutdown stop accepting new connections or closes the Listener, and then gracefully
// closes existing connections

// In the hot upgrade scenario, the Shutdown method only stops accepting new connections
// but does not close the Listener. The new Mosn can still handle some newly established
// connections after taking over the Listener.
//
// In non-hot upgrade scenarios, the Shutdown method will first close the Listener to
// directly reject the establishment of new connections. This is because if only new
// connection processing is stopped, the requests on these connections cannot be processed in the future.
func (l *listener) Shutdown(lctx context.Context) error {
	// #2220: An exception may occur when a new connection in the gracefully shutdown during non-hot upgrade
	// If current state is upgrading, it means old Mosn exits after new Mosn starts successfully.
	if stagemanager.GetState() == stagemanager.Upgrading {
		// If it is upgrade, stop accept new connection only,
		changed, err := l.stopAccept()
		if changed {
			l.cb.OnShutdown()
		}
		return err
	} else {
		// If it is not upgrade, close the listener first to avoid new connection establishment.
		// These newly established connections will not be processed by mosn before listener closed, and the requests on the connection will timeout.

		// Close listener should not close existsing connections.
		err := l.Close(lctx)

		if l.bindToPort {
			// Close the existing connections if it support graceful close.
			l.cb.OnShutdown()
		}

		return err
	}
}

// stopAccept just stop accepting new connections
func (l *listener) stopAccept() (changed bool, err error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.state == ListenerClosed || l.state == ListenerStopped {
		return
	}
	l.state = ListenerStopped
	changed = true

	if !l.bindToPort {
		return
	}
	err = l.setDeadline(time.Now())
	return
}

func (l *listener) setDeadline(t time.Time) error {
	var err error
	switch l.network {
	case "udp":
		if l.packetConn == nil {
			return errors.New("setDeadline: packetConn is nil")
		}
		err = l.packetConn.SetDeadline(t)
	case "unix":
		if l.rawl == nil {
			return errors.New("setDeadline: raw listener is nil")
		}
		err = l.rawl.(*net.UnixListener).SetDeadline(t)
	case "tcp":
		if l.rawl == nil {
			return errors.New("setDeadline: raw listener is nil")
		}
		err = l.rawl.(*net.TCPListener).SetDeadline(t)
	}

	return err
}

func (l *listener) ListenerTag() uint64 {
	return l.listenerTag
}

func (l *listener) SetListenerTag(tag uint64) {
	l.listenerTag = tag
}

func (l *listener) ListenerFile() (*os.File, error) {
	if !l.bindToPort {
		return nil, syscall.EINVAL
	}

	switch l.network {
	case "udp":
		if l.packetConn == nil {
			return nil, errors.New("ListenerFile: packetConn is nil")
		}
		return l.packetConn.(*net.UDPConn).File()
	case "unix":
		if l.rawl == nil {
			return nil, errors.New("ListenerFile: raw listener is nil")
		}
		return l.rawl.(*net.UnixListener).File()
	case "tcp":
		if l.rawl == nil {
			return nil, errors.New("ListenerFile: raw listener is nil")
		}
		return l.rawl.(*net.TCPListener).File()
	}

	return nil, errors.New("not support this network " + l.network)
}

func (l *listener) PerConnBufferLimitBytes() uint32 {
	return l.perConnBufferLimitBytes
}

func (l *listener) SetPerConnBufferLimitBytes(limitBytes uint32) {
	l.perConnBufferLimitBytes = limitBytes
}

func (l *listener) SetListenerCallbacks(cb types.ListenerEventListener) {
	l.cb = cb
}

func (l *listener) GetListenerCallbacks() types.ListenerEventListener {
	return l.cb
}

func (l *listener) SetOriginalDstType(t v2.OriginalDstType) {
	l.OriginalDst = t
}

func (l *listener) GetOriginalDstType() v2.OriginalDstType {
	return l.OriginalDst
}

func (l *listener) IsOriginalDst() bool {
	if l.OriginalDst == v2.REDIRECT || l.OriginalDst == v2.TPROXY {
		return true
	} else {
		return false
	}
}

func (l *listener) Close(lctx context.Context) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.state == ListenerClosed {
		return nil
	}

	l.state = ListenerClosed

	if !l.bindToPort {
		return nil
	}

	if l.rawl != nil {
		if l.cb != nil {
			l.cb.OnClose()
		}
		return l.rawl.Close()
	}
	if l.packetConn != nil {
		if l.cb != nil {
			l.cb.OnClose()
		}
		return l.packetConn.Close()
	}
	return nil
}

func (l *listener) listen(lctx context.Context) error {
	var err error
	var rawl net.Listener
	var rconn net.PacketConn

	if l.localAddress == nil {
		return errors.New("listener local addr is nil")
	}

	switch l.network {
	case "udp":
		lc := net.ListenConfig{}
		if rconn, err = lc.ListenPacket(context.Background(), l.network, l.localAddress.String()); err != nil {
			return err
		}
		l.packetConn = rconn
	case "unix":
		// delete the unix socket file prior to binding, only 'failsafe' way
		if err := os.Remove(l.localAddress.String()); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove unix socket file: %v", err)
		}
		if rawl, err = net.Listen("unix", l.localAddress.String()); err != nil {
			return err
		}
		l.rawl = rawl
	case "tcp":
		if rawl, err = net.Listen("tcp", l.localAddress.String()); err != nil {
			return err
		}

		if l.OriginalDst == v2.TPROXY {
			rawConn, err := rawl.(*net.TCPListener).SyscallConn()
			if err != nil {
				return err
			}
			var controlError error
			if err := rawConn.Control(func(fd uintptr) {
				if err = syscall.SetsockoptInt(int(fd), SOL_IP, IP_TRANSPARENT, 1); err != nil {
					controlError = fmt.Errorf(
						"failed to set socket opt IP_TRANSPARENT for listener %s: %s",
						l.localAddress.String(), err.Error(),
					)
					log.DefaultLogger.Errorf(controlError.Error())
				}
			}); err != nil {
				return err
			}

			if controlError != nil {
				return controlError
			}
		}
		l.rawl = rawl
	}

	return nil
}

func (l *listener) accept(lctx context.Context) error {
	rawc, err := l.rawl.Accept()

	if err != nil {
		return err
	}

	// TODO: use thread pool
	utils.GoWithRecover(func() {
		if l.cb != nil {
			l.cb.OnAccept(rawc, l.IsOriginalDst(), nil, nil, nil, nil)
		}
	}, nil)

	return nil
}
