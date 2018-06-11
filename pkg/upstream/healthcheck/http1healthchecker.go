package healthcheck

import (
	"time"
)

func StartHttpHealthCheck(tickInterval time.Duration, timerTimeout time.Duration, checkPath string,
	intervalCB func(path string, successCall func()), timeoutCB func()) {

	shc := Http1HealthCheck{
		checkPath:  checkPath,
		intervalCB: intervalCB,
		timeout:    timerTimeout,
		interval:   tickInterval,
	}

	shc.intervalTimer = newTimer(shc.OnInterval)
	shc.timeoutTimer = newTimer(timeoutCB)

	shc.Start()
}

type Http1HealthCheck struct {
	intervalTimer *timer
	timeoutTimer  *timer
	checkPath     string
	intervalCB    func(string string, h1f func())
	timeout       time.Duration
	interval      time.Duration
}

func (shc *Http1HealthCheck) Start() {
	shc.OnInterval()
}

func (shc *Http1HealthCheck) OnInterval() {
	// call app's function
	shc.intervalCB(shc.checkPath, shc.OnTimeoutTimerRest)
	shc.timeoutTimer.start(shc.timeout)
}

func (shc *Http1HealthCheck) OnTimeoutTimerRest() {
	shc.timeoutTimer.stop()
	shc.intervalTimer.start(shc.interval)
}
