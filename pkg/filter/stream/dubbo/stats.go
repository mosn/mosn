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

package dubbo

import (
	"sync"

	gometrics "github.com/rcrowley/go-metrics"
	"mosn.io/mosn/pkg/istio"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
)

var (
	ServiceInfo  = "request_total"
	ResponseSucc = "response_succ_total"
	ResponseFail = "response_fail_total"
	RequestTime  = "request_time"

	listenerKey = "listener"
	serviceKey  = "service"
	methodKey   = "method"
	subsetKey   = "subset"

	podSubsetKey = ""
	metricPre    = "mosn"
)

var (
	l            sync.RWMutex
	statsFactory = make(map[string]*Stats)
)

type Stats struct {
	RequestServiceInfo gometrics.Counter
	ResponseSucc       gometrics.Counter
	ResponseFail       gometrics.Counter
}

func getStats(listener, service, method string) *Stats {
	key := service + "-" + method

	l.RLock()
	s, ok := statsFactory[key]
	l.RUnlock()
	if ok {
		return s
	}

	l.Lock()
	defer l.Unlock()
	if s, ok = statsFactory[key]; ok {
		return s
	}

	lables := map[string]string{
		listenerKey: listener,
		serviceKey:  service,
		methodKey:   method,
	}
	if podSubsetKey != "" {
		pl := istio.GetPodLabels()
		if pl[podSubsetKey] != "" {
			lables[subsetKey] = pl[podSubsetKey]
		}
	}

	mts, err := metrics.NewMetrics(metricPre, lables)
	if err != nil {
		log.DefaultLogger.Errorf("create metrics fail: labels:%v, err: %v", lables, err)
		statsFactory[key] = nil
		return nil
	}

	s = &Stats{
		RequestServiceInfo: mts.Counter(ServiceInfo),
		ResponseSucc:       mts.Counter(ResponseSucc),
		ResponseFail:       mts.Counter(ResponseFail),
	}
	statsFactory[key] = s
	return s
}
