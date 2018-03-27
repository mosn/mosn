package healthcheck

import "time"

type timer struct {
	callback func()
	stopped  bool
	stopChan chan bool
}

func newTimer(callback func()) *timer {
	return &timer{
		callback: callback,
		stopChan: make(chan bool),
	}
}

func (t *timer) start(interval time.Duration) {
	go func() {
		select {
		case <-time.After(interval):
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
