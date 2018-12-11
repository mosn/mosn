package healthcheck

import (
	"github.com/alipay/sofa-mosn/pkg/types"
)

// checker is a wrapper of types.HealthCheckSession for health check
type checker struct {
	Session       types.HealthCheckSession
	Host          types.Host
	HealthChecker *healthChecker
	//
	resp          chan checkResponse
	timeout       chan bool
	checkID       uint64
	stop          chan struct{}
	checkTimer    *timer
	checkTimeout  *timer
	unHealthCount uint32
	healthCount   uint32
}

type checkResponse struct {
	ID      uint64
	Healthy bool
}

func newChecker(s types.HealthCheckSession, h types.Host, hc *healthChecker) *checker {
	c := &checker{
		Session:       s,
		Host:          h,
		HealthChecker: hc,
		resp:          make(chan checkResponse),
		timeout:       make(chan bool),
		stop:          make(chan struct{}),
	}
	c.checkTimer = newTimer(c.OnCheck)
	c.checkTimeout = newTimer(c.OnTimeout)
	return c
}

func (c *checker) Start() {
	defer func() {
		c.checkTimer.stop()
		c.checkTimeout.stop()
	}()
	c.checkTimer.start(c.HealthChecker.getCheckInterval())
	for {
		select {
		case <-c.stop:
			return
		default:
			// prepare a check
			c.checkID++
			select {
			case <-c.stop:
				return
			case resp := <-c.resp:
				// check the ID, ignore the timeout resp
				if resp.ID == c.checkID {
					c.checkTimeout.stop()
					if resp.Healthy {
						c.HandleSuccess()
					} else {
						c.HandleFailure(types.FailureActive)
					}
					c.checkTimer.start(c.HealthChecker.getCheckInterval())
				}
			case <-c.timeout:
				c.checkTimer.stop()
				c.Session.OnTimeout() // session timeout callbacks
				c.HandleFailure(types.FailureNetwork)
				c.checkTimer.start(c.HealthChecker.getCheckInterval())
			}
		}
	}
}

func (c *checker) Stop() {
	close(c.stop)
}

func (c *checker) HandleSuccess() {
	c.unHealthCount = 0
	changed := false
	if c.Host.ContainHealthFlag(types.FAILED_ACTIVE_HC) {
		c.healthCount++
		if c.healthCount == c.HealthChecker.healthyThreshold {
			changed = true
			c.Host.ClearHealthFlag(types.FAILED_ACTIVE_HC)
		}
	}
	c.HealthChecker.incHealthy(c.Host, changed)
}

func (c *checker) HandleFailure(reason types.FailureType) {
	c.healthCount = 0
	changed := false
	if !c.Host.ContainHealthFlag(types.FAILED_ACTIVE_HC) {
		c.unHealthCount++
		if c.unHealthCount == c.HealthChecker.unhealthyThreshold {
			changed = true
			c.Host.SetHealthFlag(types.FAILED_ACTIVE_HC)
		}
	}
	c.HealthChecker.decHealthy(c.Host, reason, changed)
}

func (c *checker) OnCheck() {
	// record current id
	id := c.checkID
	c.HealthChecker.stats.attempt.Inc(1)
	// start a timeout before check health
	c.checkTimeout.start(c.HealthChecker.timeout)
	c.resp <- checkResponse{
		ID:      id,
		Healthy: c.Session.CheckHealth(),
	}
}

func (c *checker) OnTimeout() {
	c.timeout <- true
}
