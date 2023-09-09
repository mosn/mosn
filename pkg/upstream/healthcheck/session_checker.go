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

package healthcheck

import (
	"context"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	ants "github.com/panjf2000/ants/v2"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

// TODO healtcheck workpool could dynamic scale base on types.Host count
var workpool *ants.Pool
var oneInitPool sync.Once

// InitCheckWorkPool init health check work pool
// In a large-scale types.Host scenario, the goroutine worker pool can reduce the number of
// goroutines used for health checking, and reduce the consumption of memory and go runtime scheduling
func InitCheckWorkPool(conf *v2.HealthCheckWorkpool) error {
	poolOptions := ants.Options{
		ExpiryDuration:   conf.ExpiryDuration,
		PreAlloc:         conf.PreAlloc,
		MaxBlockingTasks: conf.MaxBlockingTasks,
		Nonblocking:      conf.Nonblocking,
		DisablePurge:     conf.DisablePurge,
	}
	var err error
	oneInitPool.Do(func() {
		var options []ants.Option
		options = append(options, ants.WithOptions(poolOptions))
		options = append(options, ants.WithPanicHandler(func(i interface{}) {
			log.DefaultLogger.Alertf("healthcheck.session", "[upstream] [health check] [session checker] panic %v\n%s", i, string(debug.Stack()))
		}))
		workpool, err = ants.NewPool(conf.Size, options...)
	})
	if err != nil {
		return err
	}
	return nil
}

// GetWorkPool could dynamic set pool size according to the size of the types.Cluster or types.Host
func GetWorkPool() *ants.Pool {
	return workpool
}

// sessionChecker is a wrapper of types.HealthCheckSession for health check
type sessionChecker struct {
	Session       types.HealthCheckSession
	Host          types.Host
	HealthChecker *healthChecker
	//
	checkID uint64
	stop    chan struct{}
	//checkTimer    *utils.Timer
	checkTimer    atomic.Value // value is checkTimer
	unHealthCount uint32
	healthCount   uint32
	checkingState uint32 // 0: not-checking, 1: checking
	workpool      *ants.Pool
}

func newChecker(s types.HealthCheckSession, h types.Host, hc *healthChecker, workpool *ants.Pool) *sessionChecker {
	c := &sessionChecker{
		Session:       s,
		Host:          h,
		HealthChecker: hc,
		stop:          make(chan struct{}),
		workpool:      workpool,
	}
	return c
}

var firstInterval = time.Second

func (c *sessionChecker) Start() {
	t := utils.NewTimer(c.HealthChecker.initialDelay, c.putCheckTask) // TODO use time wheel
	c.checkTimer.Store(t)
}

func (c *sessionChecker) Stop() {
	close(c.stop)
	c.checkTimer.Load().(*utils.Timer).Stop()
}

func (c *sessionChecker) putCheckTask() {
	if c.workpool != nil {
		err := c.workpool.Submit(c.OnCheck)
		if err != nil {
			log.DefaultLogger.Warnf("[upstream] [health check] [session checker] [putCheckTask] "+
				"submit check task error: %+v, check id: %d", err, atomic.LoadUint64(&c.checkID))
		}
		c.checkTimer.Load().(*utils.Timer).Reset(c.HealthChecker.getCheckInterval())
		return
	}
	// new goroutime to check
	go c.OnCheck()
	c.checkTimer.Load().(*utils.Timer).Reset(c.HealthChecker.getCheckInterval())
	return
}

func (c *sessionChecker) HandleSuccess() {
	c.unHealthCount = 0
	changed := false
	if c.Host.ContainHealthFlag(api.FAILED_ACTIVE_HC) {
		c.healthCount++
		// check the threshold
		if c.healthCount == c.HealthChecker.healthyThreshold {
			changed = true
			c.Host.ClearHealthFlag(api.FAILED_ACTIVE_HC)
		}
	}
	c.HealthChecker.log(c.Host, true, changed)
	c.HealthChecker.incHealthy(c.Host, changed)
}

func (c *sessionChecker) HandleFailure(reason types.FailureType) {
	c.healthCount = 0
	changed := false
	if !c.Host.ContainHealthFlag(api.FAILED_ACTIVE_HC) {
		c.unHealthCount++
		// check the threshold
		if c.unHealthCount == c.HealthChecker.unhealthyThreshold {
			changed = true
			c.Host.SetHealthFlag(api.FAILED_ACTIVE_HC)
		}
	}
	c.HealthChecker.decHealthy(c.Host, reason, changed)
	c.HealthChecker.log(c.Host, false, changed)
}

func (c *sessionChecker) OnCheck() {
	if !atomic.CompareAndSwapUint32(&c.checkingState, 0, 1) {
		return // not concurrency check health
	}
	defer func() {
		atomic.StoreUint32(&c.checkingState, 0)
	}()
	var isStop bool
	// record current id
	id := atomic.AddUint64(&c.checkID, 1)
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[upstream] [health check] [session checker] [OnCheck] "+
			"begin new checkign, check id: %d, host: %v", id, c.Host.AddressString())
	}
	c.HealthChecker.stats.attempt.Inc(1)
	ctx, cancel := context.WithTimeout(context.Background(), c.HealthChecker.timeout)
	defer cancel()
	checkResult := c.Session.CheckHealth(ctx)

	isStop = c.isStop()
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[upstream] [health check] [session checker] [OnCheck] "+
			"check result: %v, isStop: %v, check id: %d, host: %v", checkResult, isStop, id, c.Host.AddressString())
	}
	if isStop {
		return
	}

	if checkResult {
		c.HandleSuccess()
	} else {
		select {
		case <-ctx.Done():
			c.HandleFailure(types.FailureNetwork)
		default:
			c.HandleFailure(types.FailureActive)
		}
	}
}

func (c *sessionChecker) isStop() bool {
	select {
	case <-c.stop: // avoid goroutine leak https://github.com/mosn/mosn/issues/2336
		return true
	default:
		return false
	}
}
