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
	"fmt"
	"strconv"
	"strings"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

// ReportInfo get additional report info.
type ReportInfo struct {
	responseTotalSize uint64
	requestTotalSize  uint64
	duration          time.Duration
	responseCode      int
	requestBodySize   uint64
	responseBodySize  uint64
	responseFlag      api.ResponseFlag
}

// ReportData extract HTTP data for Mixer report.
// Implemented by the environment (Envoy) and used by the library.
type ReportData interface {
	// GetReportInfo return ReportInfo
	GetReportInfo() (data ReportInfo)

	// GetDestinationIPPort return destination ip and port
	GetDestinationIPPort() (ip string, port int, err error)

	// GetDestinationUID return destination UID
	//GetDestinationUID() (uid string, err error)

	// GetResponseHeaders return response HTTP headers
	GetResponseHeaders() (headers types.HeaderMap)
}

type reportData struct {
	respHeaders       types.HeaderMap
	requestInfo       types.RequestInfo
	requestTotalSize  uint64
	responseTotalSize uint64
}

// NewReportData return ReportData
func NewReportData(respHeaders types.HeaderMap, requestInfo types.RequestInfo, requestTotalSize uint64) ReportData {
	return &reportData{
		respHeaders:       respHeaders,
		requestInfo:       requestInfo,
		requestTotalSize:  requestTotalSize,
		responseTotalSize: requestInfo.BytesSent() + respHeaders.ByteSize(),
	}
}

// GetReportInfo return ReportInfo
func (r *reportData) GetReportInfo() (data ReportInfo) {
	data.requestBodySize = r.requestInfo.BytesReceived()
	data.responseBodySize = r.requestInfo.BytesSent()
	data.requestTotalSize = r.requestTotalSize
	data.responseTotalSize = r.responseTotalSize
	data.duration = r.requestInfo.Duration()
	data.responseCode = r.requestInfo.ResponseCode()
	return
}

// GetDestinationIPPort return destination ip and port
func (r *reportData) GetDestinationIPPort() (ip string, port int, err error) {
	hostInfo := r.requestInfo.UpstreamHost()
	if hostInfo == nil {
		err = fmt.Errorf("no host info")
		return
	}

	address := hostInfo.AddressString()
	array := strings.Split(address, ":")
	if len(array) != 2 {
		err = fmt.Errorf("wrong format of address %v", address)
		return
	}
	ip = array[0]
	portStr := array[1]
	port, err = strconv.Atoi(portStr)
	return
}

// GetDestinationUID return destination UID
/*
func (r *reportData) GetDestinationUID() (uid string, err error) {
	hostInfo := r.requestInfo.UpstreamHost()
	if hostInfo == nil {
		err = fmt.Errorf("no host info")
		return
	}

	return utils.GetDestinationUID(hostInfo.Metadata())
}
*/

// GetResponseHeaders return response HTTP headers
func (r *reportData) GetResponseHeaders() (headers types.HeaderMap) {
	return r.respHeaders
}
