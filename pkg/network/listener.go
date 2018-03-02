package network

import (
	"net"
	"context"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"fmt"
)

// listener impl based on golang net package
type listener struct {
	name                                  string
	localAddress                          string
	bindToPort                            bool
	listenerTag                           uint64
	connBufferLimitBytes                  uint32
	handOffRestoredDestinationConnections bool
	cb                                    types.ListenerCallbacks
	rawl                                  *net.TCPListener
}

func NewListener(lc v2.ListenerConfig) types.Listener {
	l := &listener{
		name:                                  lc.Name,
		localAddress:                          lc.Addr,
		bindToPort:                            lc.BindToPort,
		listenerTag:                           lc.ListenerTag,
		connBufferLimitBytes:                  lc.ConnBufferLimitBytes,
		handOffRestoredDestinationConnections: lc.HandOffRestoredDestinationConnections,
	}

	return l
}

func (l *listener) Name() string {
	return l.name
}

func (l *listener) Addr() net.Addr {
	return l.rawl.Addr()
}

func (l *listener) Start(stopChan chan bool, lctx context.Context) {
	if err := l.listen(lctx); err != nil {
		// TODO: notify listener callbacks
		return
	}

	if l.bindToPort {
		for {
			select {
			case <-stopChan:
				return
			default:
				if err := l.accept(lctx); err != nil {
					if ope, ok := err.(*net.OpError); ok {
						if !(ope.Timeout() && ope.Temporary()) {
							// LOGGING
						}
					} else {
						// LOGGING
					}
				}
			}
		}
	}
}

func (l *listener) ListenerTag() uint64 {
	return l.listenerTag
}

func (l *listener) SetListenerCallbacks(cb types.ListenerCallbacks) {
	l.cb = cb
}

func (l *listener) Close(lctx context.Context) error {
	l.cb.OnClose()
	return l.rawl.Close()
}

func (l *listener) listen(lctx context.Context) error {
	var err error
	var la *net.TCPAddr

	if la, err = net.ResolveTCPAddr("tcp", l.localAddress); err != nil {
		return err
	}

	var rawl *net.TCPListener
	if rawl, err = net.ListenTCP("tcp", la); err != nil {
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
				fmt.Printf("panic %v", p)
			}
		}()

		l.cb.OnAccept(rawc, l.handOffRestoredDestinationConnections)
	}()

	return nil
}
