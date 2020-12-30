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
package msgconnpool

import (
	"context"
	"sync/atomic"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

// types.StreamEventListener
// types.ConnectionEventListener
// types.StreamConnectionEventListener
type activeClient struct {
	pool            *connpool
	connData        atomic.Value // *types.CreateConnectionData
	keepAlive       *keepAliveListener
	connectTryTimes int
}

// get connData
func (ac *activeClient) getConnData() *types.CreateConnectionData {
	// use atomic to load pointer, avoid partial pointer read
	c := ac.connData.Load()
	if cd, ok := c.(*types.CreateConnectionData); ok {
		return cd
	}

	return nil
}

// setHeartBeater set the heart beat for an active client
func (ac *activeClient) setHeartBeater(kp KeepAlive) {
	if kp == nil {
		return
	}

	kp.SaveHeartBeatFailCallback(func() {
		ac.getConnData().Connection.Close(api.NoFlush, api.LocalClose)
	})

	ac.keepAlive = &keepAliveListener{
		keepAlive: kp,
		conn:      ac.getConnData().Connection,
	}

	ac.getConnData().Connection.AddConnectionEventListener(ac.keepAlive)
}

func (ac *activeClient) clearHeartBeater() {
	// clear the previous keepAlive
	if ac.keepAlive != nil && ac.keepAlive.keepAlive != nil {
		if !atomic.CompareAndSwapUint64(&ac.keepAlive.stopped, 0, 1) {
			// this keepalive object is already stopped
			return
		}

		ac.keepAlive.keepAlive.Stop()
	}
}

// reconnect triggers connection to reconnect
func (ac *activeClient) reconnect(event api.ConnectionEvent) {
	if atomic.LoadUint64(&ac.pool.destroyed) == 1 {
		return
	}

	if ac.connectTryTimes >= ac.pool.connTryTimes {
		if log.DefaultLogger.GetLogLevel() >= log.WARN {
			log.DefaultLogger.Warnf("[connpool] retry time exceed pool config %v", ac.pool.connTryTimes)
		}
		return
	}

	utils.GoWithRecover(func() {
		time.Sleep(getBackOffDuration(ac.connectTryTimes))

		ac.pool.connectingMux.Lock()
		defer ac.pool.connectingMux.Unlock()

		ac.initConnectionLocked(string(event))

	}, func(r interface{}) {
		log.DefaultLogger.Errorf("[connpool] panic while reconnect, %v, connData: %v, event : %v, host : %v",
			r, ac.connData, event, ac.pool.Host().AddressString())
	})

}

func (ac *activeClient) initConnectionLocked(initReason string) {
	if atomic.LoadUint64(&ac.pool.destroyed) == 1 {
		return
	}

	ac.clearHeartBeater() // clear previous heartbeat if any

	ac.connectTryTimes++ // count connect try times

	// build new conn
	// must create this new conn, the same conn can only be connected once
	createConnData := ac.pool.Host().CreateConnection(context.Background())
	createConnData.Connection.AddConnectionEventListener(ac)

	// connect the new connection
	err := createConnData.Connection.Connect()

	if err != nil {
		if ac.getConnData() == nil {
			// the first time
			// atomic store, avoid partial write
			ac.connData.Store(&createConnData)
		}

		if log.DefaultLogger.GetLogLevel() >= log.WARN {
			log.DefaultLogger.Warnf("[connpool] connect failed %v times, host: %v, connData: %v, init reason: %v",
				ac.connectTryTimes, ac.pool.Host().AddressString(), ac.connData, initReason)
		}
		return
	}

	// atomic store, avoid partial write
	ac.connData.Store(&createConnData)

	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("[connpool] connect succeed after %v tries, host: %v, connData: %v, init reason: %v",
			ac.connectTryTimes, ac.pool.Host().AddressString(), ac.connData, initReason)
	}

	// init read filters and keep alive
	if ac.pool.getReadFilterAndKeepalive != nil {
		readFilterList, keepalive := ac.pool.getReadFilterAndKeepalive()

		for _, rf := range readFilterList {
			ac.getConnData().Connection.FilterManager().AddReadFilter(rf)
		}

		// set the new keepalive
		ac.setHeartBeater(keepalive)
	}

	// two scenes
	//   1. destroyed == true, there is no problem
	//   2. destroyed == false, concurrent with Destroy
	//      Destroy will Close this connection for us
	if atomic.LoadUint64(&ac.pool.destroyed) == 1 {
		ac.clearHeartBeater()
		ac.getConnData().Connection.Close(api.NoFlush, api.LocalClose)
		return
	}

	// clear retry times
	ac.connectTryTimes = 0
}

func (ac *activeClient) OnEvent(event api.ConnectionEvent) {
	switch {
	case event.IsClose():
		goto RECONNECT
	case event.ConnectFailure():
		goto RECONNECT
	default:
		return
	}

RECONNECT:
	if ac.pool.autoReconnectWhenClose {
		// auto reconnect when close
		if log.DefaultLogger.GetLogLevel() >= log.WARN {
			log.DefaultLogger.Warnf("[connpool] reconnect after event: %v, connData: %v, host: %v",
				event, ac.connData, ac.pool.Host().AddressString())
		}

		ac.reconnect(event)
	} else {
		if log.DefaultLogger.GetLogLevel() >= log.WARN {
			log.DefaultLogger.Warnf("[connpool] auto reconnect is closed, event : %v, connData : %v, clear heartbeat",
				event, ac.connData)
		}

		utils.GoWithRecover(func() {
			ac.pool.connectingMux.Lock()
			defer ac.pool.connectingMux.Unlock()
			ac.clearHeartBeater() // ensure the previous heartbeat is cleared
		}, nil)
	}
}
