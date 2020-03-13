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

package network

import (
	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

type filterManager struct {
	upstreamFilters   []*activeReadFilter
	downstreamFilters []api.WriteFilter
	conn              api.Connection
	host              api.HostInfo
}

func newFilterManager(conn api.Connection) api.FilterManager {
	return &filterManager{
		conn:              conn,
		upstreamFilters:   make([]*activeReadFilter, 0, 8),
		downstreamFilters: make([]api.WriteFilter, 0, 8),
	}
}

func (fm *filterManager) AddReadFilter(rf api.ReadFilter) {
	newArf := &activeReadFilter{
		filter:        rf,
		filterManager: fm,
	}

	rf.InitializeReadFilterCallbacks(newArf)
	fm.upstreamFilters = append(fm.upstreamFilters, newArf)
}

func (fm *filterManager) AddWriteFilter(wf api.WriteFilter) {
	fm.downstreamFilters = append(fm.downstreamFilters, wf)
}

func (fm *filterManager) ListReadFilter() []api.ReadFilter {
	var readFilters []api.ReadFilter

	for _, uf := range fm.upstreamFilters {
		readFilters = append(readFilters, uf.filter)
	}

	return readFilters
}

func (fm *filterManager) ListWriteFilters() []api.WriteFilter {
	return fm.downstreamFilters
}

func (fm *filterManager) InitializeReadFilters() bool {
	if len(fm.upstreamFilters) == 0 {
		return false
	}

	fm.onContinueReading(nil)
	return true
}

func (fm *filterManager) onContinueReading(filter *activeReadFilter) {
	var index int
	var uf *activeReadFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(fm.upstreamFilters); index++ {
		uf = fm.upstreamFilters[index]
		uf.index = index

		if !uf.initialized {
			uf.initialized = true

			status := uf.filter.OnNewConnection()

			if status == api.Stop {
				return
			}
		}

		buf := fm.conn.GetReadBuffer()

		if buf != nil && buf.Len() > 0 {
			status := uf.filter.OnData(buf)

			if status == api.Stop {
				//fm.conn.Write("your data")
				return
			}
		}
	}
}

func (fm *filterManager) OnRead() {
	fm.onContinueReading(nil)
}

func (fm *filterManager) OnWrite(buf []buffer.IoBuffer) api.FilterStatus {
	for _, df := range fm.downstreamFilters {
		status := df.OnWrite(buf)

		if status == api.Stop {
			return api.Stop
		}
	}

	return api.Continue
}

// as a ReadFilterCallbacks
type activeReadFilter struct {
	index         int
	filter        api.ReadFilter
	filterManager *filterManager
	initialized   bool
}

func (arf *activeReadFilter) Connection() api.Connection {
	return arf.filterManager.conn
}

func (arf *activeReadFilter) ContinueReading() {
	arf.filterManager.onContinueReading(arf)
}

func (arf *activeReadFilter) UpstreamHost() api.HostInfo {
	return arf.filterManager.host
}

func (arf *activeReadFilter) SetUpstreamHost(upstreamHost api.HostInfo) {
	arf.filterManager.host = upstreamHost
}
