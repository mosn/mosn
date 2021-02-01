// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package circuitbreaker implements the circuit breaker.
//
// Sentinel circuit breaker module converts each Rule into a CircuitBreaker. Each CircuitBreaker has its own statistical structure.
//
// Sentinel circuit breaker module supports three strategies:
//
//  1. SlowRequestRatio: the ratio of slow response time entry(entry's response time is great than max slow response time) exceeds the threshold. The following entry to resource will be broken.
//                       In SlowRequestRatio strategy, user must set max response time.
//  2. ErrorRatio: the ratio of error entry exceeds the threshold. The following entry to resource will be broken.
//  3. ErrorCount: the number of error entry exceeds the threshold. The following entry to resource will be broken.
//
// Sentinel circuit breaker is implemented based on state machines. There are three state:
//
//  1. Closed: all entries could pass checking.
//  2. Open: the circuit breaker is broken, all entries are blocked. After retry timeout, circuit breaker switches state to Half-Open and allows one entry to probe whether the resource returns to its expected state.
//  3. Half-Open: the circuit breaker is in a temporary state of probing, only one entry is allowed to access resource, others are blocked.
//
// Sentinel circuit breaker provides the listener to listen on the state changes.
//
//  type StateChangeListener interface {
//  	OnTransformToClosed(prev State, rule Rule)
//
//  	OnTransformToOpen(prev State, rule Rule, snapshot interface{})
//
//  	OnTransformToHalfOpen(prev State, rule Rule)
//  }
//
// Here is the example code to use circuit breaker:
//
//  type stateChangeTestListener struct {}
//
//  func (s *stateChangeTestListener) OnTransformToClosed(prev circuitbreaker.State, rule circuitbreaker.Rule) {
//  	fmt.Printf("rule.steategy: %+v, From %s to Closed, time: %d\n", rule.Strategy, prev.String(), util.CurrentTimeMillis())
//  }
//
//  func (s *stateChangeTestListener) OnTransformToOpen(prev circuitbreaker.State, rule circuitbreaker.Rule, snapshot interface{}) {
//  	fmt.Printf("rule.steategy: %+v, From %s to Open, snapshot: %.2f, time: %d\n", rule.Strategy, prev.String(), snapshot, util.CurrentTimeMillis())
//  }
//
//  func (s *stateChangeTestListener) OnTransformToHalfOpen(prev circuitbreaker.State, rule circuitbreaker.Rule) {
//  	fmt.Printf("rule.steategy: %+v, From %s to Half-Open, time: %d\n", rule.Strategy, prev.String(), util.CurrentTimeMillis())
//  }
//
//  func main() {
//  	err := sentinel.InitDefault()
//  	if err != nil {
//  		log.Fatal(err)
//  	}
//  	ch := make(chan struct{})
//  	// Register a state change listener so that we could observer the state change of the internal circuit breaker.
//  	circuitbreaker.RegisterStateChangeListeners(&stateChangeTestListener{})
//
//  	_, err = circuitbreaker.LoadRules([]*circuitbreaker.Rule{
//  	// Statistic time span=10s, recoveryTimeout=3s, slowRtUpperBound=50ms, maxSlowRequestRatio=50%
//  		{
//  			Resource:         "abc",
//  			Strategy:         circuitbreaker.SlowRequestRatio,
//  			RetryTimeoutMs:   3000,
//  			MinRequestAmount: 10,
//  			StatIntervalMs:   10000,
//  			MaxAllowedRtMs:   50,
//  			Threshold:        0.5,
//  		},
//  		// Statistic time span=10s, recoveryTimeout=3s, maxErrorRatio=50%
//  		{
//  			Resource:         "abc",
//  			Strategy:         circuitbreaker.ErrorRatio,
//  			RetryTimeoutMs:   3000,
//  			MinRequestAmount: 10,
//  			StatIntervalMs:   10000,
//  			Threshold:        0.5,
//  		},
//  	})
//  	if err != nil {
//  		log.Fatal(err)
//  	}
//
//  	fmt.Println("Sentinel Go circuit breaking demo is running. You may see the pass/block metric in the metric log.")
//  	go func() {
//  		for {
//  			e, b := sentinel.Entry("abc")
//  			if b != nil {
//  				//fmt.Println("g1blocked")
//  				time.Sleep(time.Duration(rand.Uint64()%20) * time.Millisecond)
//  			} else {
//  				if rand.Uint64()%20 > 9 {
//  					// Record current invocation as error.
//  					sentinel.TraceError(e, errors.New("biz error"))
//  				}
//  				//fmt.Println("g1passed")
//  				time.Sleep(time.Duration(rand.Uint64()%80+10) * time.Millisecond)
//  				e.Exit()
//  			}
//  		}
//		}()
//
//  	go func() {
//  		for {
//  			e, b := sentinel.Entry("abc")
//  			if b != nil {
//  				//fmt.Println("g2blocked")
//  				time.Sleep(time.Duration(rand.Uint64()%20) * time.Millisecond)
//  			} else {
//  				//fmt.Println("g2passed")
//  				time.Sleep(time.Duration(rand.Uint64()%80) * time.Millisecond)
//  				e.Exit()
//  			}
//  		}
//  	}()
//  	<-ch
//  }
//
package circuitbreaker
