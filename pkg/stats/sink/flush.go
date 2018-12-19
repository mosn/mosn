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

package sink

import (
	"time"

	"github.com/alipay/sofa-mosn/pkg/stats"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var defaultFlushInteval = time.Second

func StartFlush(sinks []types.MetricsSink, interval time.Duration) {
	if interval <= 0 {
		interval = defaultFlushInteval
	}

	for _ = range time.Tick(interval) {
		allRegs := stats.GetAllRegistries()
		// flush each reg to all sinks
		for _, reg := range allRegs {
			for _, sink := range sinks {
				sink.Flush(reg)
			}
		}
	}
}
