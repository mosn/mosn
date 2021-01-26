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

package regulator

import (
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
)

func Test_WorkPool(t *testing.T) {
	//
	workPool := NewDefaultWorkPool(10)

	//
	config := &v2.FaultToleranceFilterConfig{
		Enabled:               false,
		ExceptionTypes:        nil,
		TimeWindow:            5000,
		LeastWindowCount:      0,
		ExceptionRateMultiple: 0,
		MaxIpCount:            0,
		MaxIpRatio:            0,
		RecoverTime:           5000,
	}

	//
	go func() {
		for i := 0; i < 15; i++ {
			model_1 := NewMeasureModel("1"+"@"+"i", config)
			model_2 := NewMeasureModel("2"+"@"+"i", config)
			model_3 := NewMeasureModel("3"+"@"+"i", config)
			go func() {
				workPool.Schedule(model_1)
				workPool.Schedule(model_2)
				workPool.Schedule(model_3)
			}()
		}
	}()
	go func() {
		for i := 0; i < 15; i++ {
			model_4 := NewMeasureModel("4"+"@"+"i", config)
			model_5 := NewMeasureModel("5"+"@"+"i", config)
			model_6 := NewMeasureModel("6"+"@"+"i", config)
			go func() {
				workPool.Schedule(model_4)
				workPool.Schedule(model_5)
				workPool.Schedule(model_6)
			}()
		}
	}()
	go func() {
		for i := 0; i < 15; i++ {
			model_7 := NewMeasureModel("7"+"@"+"i", config)
			model_8 := NewMeasureModel("8"+"@"+"i", config)
			model_9 := NewMeasureModel("9"+"@"+"i", config)
			go func() {
				workPool.Schedule(model_7)
				workPool.Schedule(model_8)
				workPool.Schedule(model_9)
			}()
		}
	}()

	//
	time.Sleep(3 * time.Second)

	//
	if workPool.index == 135 {
		t.Log("Test_WorkPool Succeed")
	} else {
		t.Error("Test_WorkPool Failed")
	}
	//
	if value, ok := workPool.workers.Load("6"); ok {
		worker := value.(*WorkGoroutine)
		count := 0
		worker.tasks.Range(func(key, value interface{}) bool {
			count++
			return true
		})
		if count == 0 {
			t.Error("Test_WorkPool Failed")
		}
	} else {
		t.Error("Test_WorkPool Failed")
	}
}
