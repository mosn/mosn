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

package router

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/variable"
	"regexp"
	"strconv"
	"strings"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

var variableValueRegexp = regexp.MustCompile(`%.+%`)

func getHeaderFormatter(value string, append bool) headerFormatter {
	if variableValueRegexp.MatchString(value) {
		variableName := strings.Trim(value, "%")
		if _, err := variable.Check(variableName); err != nil {
			if log.DefaultLogger.GetLogLevel() >= log.WARN {
				log.DefaultLogger.Warnf("invalid variable header: %s", value)
			}
			return nil
		}
		return &variableHeaderFormatter{
			isAppend:     append,
			variableName: variableName,
		}
	}
	// doesn't match variable format but have %
	if strings.Index(value, "%") != -1 {
		if log.DefaultLogger.GetLogLevel() >= log.WARN {
			log.DefaultLogger.Warnf("invalid variable header: %s", value)
		}
		return nil
	}
	return &plainHeaderFormatter{
		isAppend:    append,
		staticValue: value,
	}
}

type plainHeaderFormatter struct {
	isAppend    bool
	staticValue string
}

func (f *plainHeaderFormatter) append() bool {
	return f.isAppend
}

func (f *plainHeaderFormatter) format(_ types.HeaderMap, _ types.RequestInfo) string {
	return f.staticValue
}

type variableHeaderFormatter struct {
	isAppend     bool
	variableName string
}

func (v *variableHeaderFormatter) format(headers types.HeaderMap, requestInfo api.RequestInfo) string {
	var (
		prefix   string
		isPrefix bool
	)
	if strings.HasPrefix(v.variableName, types.VarPrefixReqHeader) {
		prefix = types.VarPrefixReqHeader
		isPrefix = true
	} else if strings.HasPrefix(v.variableName, types.VarPrefixRespHeader) {
		prefix = types.VarPrefixRespHeader
		isPrefix = true
	}
	if isPrefix {
		headerName := strings.TrimPrefix(v.variableName, prefix)
		ret, _ := headers.Get(headerName)
		return ret
	}
	switch v.variableName {
	case types.VarStartTime:
		return requestInfo.StartTime().String()
	case types.VarRequestReceivedDuration:
		return requestInfo.RequestReceivedDuration().String()
	case types.VarResponseReceivedDuration:
		return requestInfo.ResponseReceivedDuration().String()
	case types.VarRequestFinishedDuration:
		return requestInfo.RequestFinishedDuration().String()
	case types.VarBytesSent:
		return strconv.FormatUint(requestInfo.BytesSent(), 10)
	case types.VarBytesReceived:
		return strconv.FormatUint(requestInfo.BytesReceived(), 10)
	case types.VarProtocol:
		return string(requestInfo.Protocol())
	case types.VarResponseCode:
		return strconv.Itoa(requestInfo.ResponseCode())
	case types.VarDuration:
		return requestInfo.Duration().String()
	case types.VarUpstreamLocalAddress:
		return requestInfo.UpstreamLocalAddress()
	case types.VarDownstreamLocalAddress:
		if requestInfo.DownstreamLocalAddress() != nil {
			return requestInfo.DownstreamLocalAddress().String()
		}
		return ""
	case types.VarDownstreamRemoteAddress:
		if requestInfo.DownstreamRemoteAddress() != nil {
			return requestInfo.DownstreamRemoteAddress().String()
		}
		return ""
	case types.VarUpstreamHost:
		if requestInfo.UpstreamHost() != nil {
			return requestInfo.UpstreamHost().AddressString()
		}
		return ""
	default:
		return ""
	}
}

func (v *variableHeaderFormatter) append() bool {
	return v.isAppend
}
