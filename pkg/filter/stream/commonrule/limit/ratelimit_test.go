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

package limit

import (
	"math"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/metrix"
	"github.com/alipay/sofa-mosn/pkg/log"
)

func TestRateLimiter_TryAcquire(t *testing.T) {
	limiter, err := NewRateLimiter(0, 1000, 1.0)
	if err != nil {
		t.Errorf("%v", err)
	}
	res := limiter.TryAcquire()
	if res {
		t.Errorf("false")
	} else {
		t.Log("ok")
	}
}

func TestRateLimiter_TryAcquire1(t *testing.T) {
	limiter, err := NewRateLimiter(1, 1000, 1.0)
	if err != nil {
		t.Errorf("%v", err)
	}

	total := 0
	success := 0
	ticker := metrix.NewTicker(func() {
		total++
		res := limiter.TryAcquire()
		if res {
			success++
		}
	})
	ticker.Start(time.Millisecond * 100)
	time.Sleep(5 * time.Second)
	ticker.Stop()
	if math.Abs(float64(success-5)) > 1 {
		t.Errorf("false, success=%d", success)
	} else {
		t.Log("total = ", total)
		t.Log("success = ", success)
	}
}

func TestRateLimiter_TryAcquire2(t *testing.T) {
	maxAllowsList := []int64{1, 20, 500}
	periodMsList := []int64{1000, 3000}
	intervals := []time.Duration{500, 10}
	sleepSec := time.Duration(3)

	log.DefaultLogger.Infof("start")
	for _, maxAllow := range maxAllowsList {
		for _, periodMs := range periodMsList {
			for _, interval := range intervals {
				log.DefaultLogger.Infof("maxAllow=%d, periodMs=%d, interval=%d", maxAllow, periodMs, interval)
				limiter, _ := NewRateLimiter(maxAllow, periodMs, 1.0)

				total := 0
				success := 0
				ticker := metrix.NewTicker(func() {
					total++
					res := limiter.TryAcquire()
					if res {
						success++
					}
				})
				ticker.Start(time.Millisecond * interval)
				time.Sleep(sleepSec * time.Second)
				ticker.Stop()
				threshold := math.Min(float64(maxAllow*1000*int64(sleepSec))/float64(periodMs), float64(total))
				log.DefaultLogger.Infof("total=%d, success=%d, threshold=%f", total, success, threshold)
				if math.Abs(float64(success)-threshold) > 1 {
					t.Errorf("false, success=%d", success)
				}
			}
		}
	}
	t.Log("end")
}
