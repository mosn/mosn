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

package grpcmetric

import (
	"sync"

	gometrics "github.com/rcrowley/go-metrics"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
)

var (
	serviceReqNum = "request_total"
	responseSucc  = "response_succ_total"
	responseFail  = "response_fail_total"
	serviceKey    = "service"
	metricPre     = "grpc"
)

type stats struct {
	requestServiceTotal gometrics.Counter
	responseSuccess     gometrics.Counter
	responseFail        gometrics.Counter
}

type state struct {
	mux          sync.RWMutex
	statsFactory map[string]*stats
}

func newState() *state {
	return &state{statsFactory: make(map[string]*stats)}
}

func (s *state) getStats(service string) *stats {
	key := service
	s.mux.RLock()
	stat, ok := s.statsFactory[key]
	s.mux.RUnlock()
	if ok {
		return stat
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	if stat, ok = s.statsFactory[key]; ok {
		return stat
	}
	labels := map[string]string{
		serviceKey: key,
	}
	mts, err := metrics.NewMetrics(metricPre, labels)
	if err != nil {
		log.DefaultLogger.Errorf("create metrics fail: labels:%v, err: %v", labels, err)
		s.statsFactory[key] = nil
		return nil
	}

	stat = &stats{
		requestServiceTotal: mts.Counter(serviceReqNum),
		responseSuccess:     mts.Counter(responseSucc),
		responseFail:        mts.Counter(responseFail),
	}
	s.statsFactory[key] = stat
	return stat
}
