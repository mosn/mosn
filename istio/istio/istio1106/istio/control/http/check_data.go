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
	"encoding/base64"

	"mosn.io/mosn/istio/istio1106/istio/utils"
	"mosn.io/mosn/pkg/types"
)

// CheckData extract HTTP data for Mixer check
type CheckData interface {
	// Find "x-istio-attributes" HTTP header.
	// If found, base64 decode its value,  pass it out
	ExtractIstioAttributes() (data string, ret bool)

	// Get downstream tcp connection ip and port.
	GetSourceIPPort() (ip string, port int32, ret bool)
}

type checkData struct {
	reqHeaders  types.HeaderMap
	requestInfo types.RequestInfo
}

// NewCheckData return checkData
func NewCheckData(reqHeaders types.HeaderMap, requestInfo types.RequestInfo) CheckData {
	return &checkData{
		reqHeaders:  reqHeaders,
		requestInfo: requestInfo,
	}
}

// ExtractIstioAttributes Find "x-istio-attributes" HTTP header.
// If found, base64 decode its value,  pass it out
func (c *checkData) ExtractIstioAttributes() (data string, ret bool) {
	val, ret := c.reqHeaders.Get(utils.KIstioAttributeHeader)
	if ret {
		d, _ := base64.StdEncoding.DecodeString(val)
		data = string(d)
	}
	return
}

// GetSourceIPPort get downstream tcp connection ip and port.
func (c *checkData) GetSourceIPPort() (ip string, port int32, ret bool) {
	addr := c.requestInfo.DownstreamLocalAddress()
	if addr == nil {
		ret = false
		return
	}
	ip, port, ret = utils.GetIPPort(addr)
	return
}
