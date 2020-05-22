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

package cluster

import (
	"sync"
	"testing"

	"mosn.io/api"
)

func TestHealthFlag(t *testing.T) {
	// clear health store
	healthStore = sync.Map{}
	addr := "127.0.0.1:8080"
	testHost := &simpleHost{
		healthFlags: GetHealthFlagPointer(addr),
	}
	if !testHost.Health() {
		t.Fatal("default host is not healthy")
	}
	testHost.SetHealthFlag(api.FAILED_ACTIVE_HC)
	if testHost.Health() {
		t.Fatal("host is healthy, but expected not")
	}
	if !testHost.ContainHealthFlag(api.FAILED_ACTIVE_HC) {
		t.Fatal("host is not contain failed active")
	}
	testHost.SetHealthFlag(api.FAILED_OUTLIER_CHECK)
	if testHost.Health() {
		t.Fatal("host is healthy, but expected not")
	}
	if !testHost.ContainHealthFlag(api.FAILED_OUTLIER_CHECK) {
		t.Fatal("host is not contain failed outlier")
	}
	testHost.ClearHealthFlag(api.FAILED_OUTLIER_CHECK)
	if testHost.ContainHealthFlag(api.FAILED_OUTLIER_CHECK) {
		t.Fatal("clear outlier failed")
	}
	if testHost.Health() {
		t.Fatal("host is healthy, but expected not")
	}
	testHost.ClearHealthFlag(api.FAILED_ACTIVE_HC)
	if !testHost.Health() {
		t.Fatal("host expected healthy, but not")
	}
}

func TestHealthFlagShare(t *testing.T) {
	// clear health store
	healthStore = sync.Map{}

	addr := "127.0.0.1:8080"
	testHost := &simpleHost{
		healthFlags: GetHealthFlagPointer(addr),
	}
	testHost2 := &simpleHost{
		healthFlags: GetHealthFlagPointer(addr),
	}
	testHost.SetHealthFlag(api.FAILED_ACTIVE_HC)
	if testHost2.Health() {
		t.Fatal("test host2 should be unhealthy")
	}
	// concurrency read / set
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		for i := 0; i < 100000; i++ {
			testHost.Health()
		}
		wg.Done()
	}()
	go func() {
		for i := 0; i < 100000; i++ {
			testHost2.SetHealthFlag(api.FAILED_ACTIVE_HC)
		}
		wg.Done()
	}()
	go func() {
		for i := 0; i < 100000; i++ {
			testHost.SetHealthFlag(api.FAILED_OUTLIER_CHECK)
		}
		wg.Done()
	}()
	wg.Wait()
}
