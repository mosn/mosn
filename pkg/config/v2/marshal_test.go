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

package v2

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFaultInjectMarshal(t *testing.T) {
	f := &FaultInject{
		DelayDuration: uint64(time.Second),
	}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"delay_duration":"1s"`) {
		t.Fatalf("unexpected output: %s", string(b))
	}
}

func TestDelayInjectMarshal(t *testing.T) {
	i := &DelayInject{
		Delay: time.Second,
	}
	b, err := json.Marshal(i)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"fixed_delay":"1s"`) {
		t.Fatalf("unexpected output: %s", string(b))
	}
}

func TestHealthCheckWorkpoolJsonMarshal(t *testing.T) {
	hcwp := &HealthCheckWorkpool{
		HealthCheckWorkpoolConfig: HealthCheckWorkpoolConfig{
			Size:             100,
			PreAlloc:         true,
			MaxBlockingTasks: 10,
			Nonblocking:      true,
			DisablePurge:     true,
		},
		ExpiryDuration: 10 * time.Minute,
	}

	dataMarshal, err := json.Marshal(hcwp)
	if err != nil {
		t.Errorf("marshal healthcheck workpool error: %v", err)
		return
	}
	dataMarshalStr := string(dataMarshal)
	if !strings.Contains(dataMarshalStr, `"expiry_duration":"10m0s"`) {
		t.Errorf("marshal healthcheck workpool, want expiry_duration 10m but got: %s", dataMarshalStr)
	}
}
