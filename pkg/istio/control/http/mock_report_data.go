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

package http

import (
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

type mockReportData struct {
	getReportInfo        int
	getDestinationIPPort int
	getResponseHeaders   int
}

func newMockReportData() ReportData {
	return &mockReportData{}
}

// GetReportInfo return ReportInfo
func (r *mockReportData) GetReportInfo() (data ReportInfo) {
	r.getReportInfo++
	return
}

// GetDestinationIPPort return destination ip and port
func (r *mockReportData) GetDestinationIPPort() (ip string, port int, err error) {
	ip = "127.0.0.1"
	port = 11111
	r.getDestinationIPPort++
	return
}

// GetDestinationUID return destination UID
/*
func (r *mockReportData) GetDestinationUID() (uid string, err error) {
	uid = "test"
	r.getDestinationUID++
	return
}
*/

// GetResponseHeaders return response HTTP headers
func (r *mockReportData) GetResponseHeaders() (headers types.HeaderMap) {
	r.getResponseHeaders++
	return protocol.CommonHeader(make(map[string]string, 0))
}
