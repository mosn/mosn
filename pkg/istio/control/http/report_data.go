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

	"github.com/alipay/sofa-mosn/pkg/types"
)

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