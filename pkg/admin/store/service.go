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

package store

import (
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
	"time"
)


var startService []func()

func AddStartService(f func()) {
	startService = append(startService, f)
}

func StartService() {
	time.Sleep(1 * time.Second)
	for _, f := range startService {
		go f()
	}
	startService = startService[:0]
}

var stopService []types.StopService

func AddStopService(s types.StopService) {
	stopService = append(stopService, s)
}

func StopService() {
	for _, s := range stopService {
		if err := s.Close(); err != nil {
			log.DefaultLogger.Infof("close service error: %v", err)
		}
	}
	stopService = stopService[:0]
}