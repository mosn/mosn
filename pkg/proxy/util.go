package proxy

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"time"
	"strconv"
)

func parseProxyTimeout(route types.Route, headers map[string]string) *ProxyTimeout {
	timeout := &ProxyTimeout{}
	timeout.GlobalTimeout = route.RouteRule().GlobalTimeout()
	timeout.TryTimeout = route.RouteRule().Policy().RetryPolicy().TryTimeout()

	// todo: check global timeout in request headers
	// todo: check per try timeout in request headers

	if tto, ok := headers[types.MosnTryTimeout]; ok {
		if trytimeout, err := strconv.ParseInt(tto,10,64); err == nil {
			timeout.TryTimeout = time.Duration(trytimeout)
		}
	}

	if gto, ok := headers[types.MosnGlobalTimeout]; ok {
		if globaltimeout, err := strconv.ParseInt(gto,10,64); err == nil {
			timeout.GlobalTimeout = time.Duration(globaltimeout)
		}
	}

	if timeout.TryTimeout >= timeout.GlobalTimeout {
		timeout.TryTimeout = 0
	}

	return timeout
}

type timer struct {
	callback func()
	interval time.Duration
	stopped  bool
	stopChan chan bool
}

func newTimer(callback func(), interval time.Duration) *timer {
	return &timer{
		callback: callback,
		interval: interval,
		stopChan: make(chan bool),
	}
}

func (t *timer) start() {
	go func() {
		select {
		case <-time.After(t.interval):
			t.stopped = true
			t.callback()
		case <-t.stopChan:
			t.stopped = true
			return
		}
	}()
}

func (t *timer) stop() {
	if t.stopped {
		return
	}

	t.stopChan <- true
}
