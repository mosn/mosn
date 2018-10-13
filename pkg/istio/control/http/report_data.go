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

	"github.com/alipay/sofa-mosn/pkg/istio/utils"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type ReportInfo struct {
	responseTotalSize uint64
	requestTotalSize uint64
	duration time.Duration
	responseCode uint32
	requestBodySize uint64
	responseBodySize uint64
	responseFlag types.ResponseFlag
}

type ReportData struct {
	respHeaders types.HeaderMap
	requestInfo types.RequestInfo
}

func NewReportData(respHeaders types.HeaderMap, requestInfo types.RequestInfo) *ReportData {
	return &ReportData{
		respHeaders:respHeaders,
		requestInfo:requestInfo,
	}
}

func (r *ReportData) GetReportInfo() (data ReportInfo) {
	data.requestBodySize = r.requestInfo.BytesReceived()
	data.responseBodySize = r.requestInfo.BytesSent()
	data.duration = r.requestInfo.Duration()
	data.responseCode = r.requestInfo.ResponseCode()
	return
}

func (r *ReportData) GetDestinationIpPort()(ip string, port int, err error){
	hostInfo := r.requestInfo.UpstreamHost()
	if hostInfo == nil {
		err = fmt.Errorf("no host info")
		return
	}

	address := hostInfo.Address().String()
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

func (r *ReportData) GetDestinationUID() (uid string, err error) {
	hostInfo := r.requestInfo.UpstreamHost()
	if hostInfo == nil {
		err = fmt.Errorf("no host info")
		return
	}

	return utils.GetDestinationUID(hostInfo.Metadata())
}