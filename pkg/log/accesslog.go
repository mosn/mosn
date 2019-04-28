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

package log

import (
	"strconv"
	"strings"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// RequestInfoFuncMap is a map which key is the format-key, value is the func to get corresponding string value
var (
	RequestInfoFuncMap      map[string]func(info types.RequestInfo) string
	DefaultDisableAccessLog bool
	accessLogs              []*accesslog
)

const AccessLogLen = 1 << 8

func init() {
	RequestInfoFuncMap = map[string]func(info types.RequestInfo) string{
		types.LogStartTime:                  StartTimeGetter,
		types.LogRequestReceivedDuration:    ReceivedDurationGetter,
		types.LogResponseReceivedDuration:   ResponseReceivedDurationGetter,
		types.LogRequestFinishedDuration:    RequestFinishedDurationGetter,
		types.LogBytesSent:                  BytesSentGetter,
		types.LogBytesReceived:              BytesReceivedGetter,
		types.LogProtocol:                   ProtocolGetter,
		types.LogResponseCode:               ResponseCodeGetter,
		types.LogDuration:                   DurationGetter,
		types.LogResponseFlag:               GetResponseFlagGetter,
		types.LogUpstreamLocalAddress:       UpstreamLocalAddressGetter,
		types.LogDownstreamLocalAddress:     DownstreamLocalAddressGetter,
		types.LogDownstreamRemoteAddress:    DownstreamRemoteAddressGetter,
		types.LogUpstreamHostSelectedGetter: UpstreamHostSelectedGetter,
	}
	accessLogs = []*accesslog{}
}

func DisableAllAccessLog() {
	DefaultDisableAccessLog = true
	for _, lg := range accessLogs {
		lg.logger.Toggle(true)
	}
}

// types.AccessLog
type accesslog struct {
	output    string
	filter    types.AccessLogFilter
	formatter types.AccessLogFormatter
	logger    *Logger
}

// NewAccessLog
func NewAccessLog(output string, filter types.AccessLogFilter,
	format string) (types.AccessLog, error) {
	lg, err := GetOrCreateLogger(output)
	if err != nil {
		return nil, err
	}
	l := &accesslog{
		output:    output,
		filter:    filter,
		formatter: NewAccessLogFormatter(format),
		logger:    lg,
	}
	if DefaultDisableAccessLog {
		lg.Toggle(true) // disable accesslog by default
	}
	// save all access logs
	accessLogs = append(accessLogs, l)

	return l, nil
}

func (l *accesslog) Log(reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) {
	// return directly
	if l.logger.disable {
		return
	}
	if l.filter != nil {
		if !l.filter.Decide(reqHeaders, requestInfo) {
			return
		}
	}

	buf := buffer.GetIoBuffer(AccessLogLen)
	l.formatter.Format(buf, reqHeaders, respHeaders, requestInfo)
	// delete first " "
	if buf.Len() > 0 {
		buf.Drain(1)
	}
	buf.WriteString("\n")
	l.logger.Print(buf, true)
}

// types.AccessLogFormatter
type accesslogformatter struct {
	formatters []types.AccessLogFormatter
}

// NewAccessLogFormatter
func NewAccessLogFormatter(format string) types.AccessLogFormatter {
	if format == "" {
		format = types.DefaultAccessLogFormat
	}

	return &accesslogformatter{
		formatters: formatToFormatter(format),
	}
}

func (f *accesslogformatter) Format(buf types.IoBuffer, reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) {
	for _, formatter := range f.formatters {
		formatter.Format(buf, reqHeaders, respHeaders, requestInfo)
	}
}

// types.AccessLogFormatter
type simpleRequestInfoFormatter struct {
	reqInfoFunc []func(info types.RequestInfo) string
}

// Format request info headers
func (f *simpleRequestInfoFormatter) Format(buf types.IoBuffer, reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) {
	// todo: map fieldName to field vale string
	if f.reqInfoFunc == nil {
		DefaultLogger.Debugf("No ReqInfo Format Keys Input")
		return
	}

	for _, vFunc := range f.reqInfoFunc {
		buf.WriteString(" ")
		s := vFunc(requestInfo)
		if s == "" {
			s = "-"
		}
		buf.WriteString(s)
	}
}

// types.AccessLogFormatter
type simpleReqHeadersFormatter struct {
	reqHeaderFormat []string
}

// Format request headers format
func (f *simpleReqHeadersFormatter) Format(buf types.IoBuffer, reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) {
	if f.reqHeaderFormat == nil {
		DefaultLogger.Debugf("No ReqHeaders Format Keys Input")
		return
	}

	for _, key := range f.reqHeaderFormat {
		if v, ok := reqHeaders.Get(key); ok {
			buf.WriteString(" ")
			buf.WriteString(types.ReqHeaderPrefix)
			if v == "" {
				v = "-"
			}
			buf.WriteString(v)
		} else {
			//DefaultLogger.Debugf("Invalid reqHeaders format keys when print access log: %s", key)
		}
	}
}

// types.AccessLogFormatter
type simpleRespHeadersFormatter struct {
	respHeaderFormat []string
}

// Format response headers format
func (f *simpleRespHeadersFormatter) Format(buf types.IoBuffer, reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) {
	if f.respHeaderFormat == nil {
		DefaultLogger.Debugf("No RespHeaders Format Keys Input")
		return
	}

	for _, key := range f.respHeaderFormat {
		if respHeaders != nil {
			if v, ok := respHeaders.Get(key); ok {
				buf.WriteString(" ")
				buf.WriteString(types.RespHeaderPrefix)
				if v == "" {
					v = "-"
				}
				buf.WriteString(v)
			} else {
				//DefaultLogger.Debugf("Invalid RespHeaders Format Keys:%s", key)
			}
		}
	}
}

// format to formatter by parsing format
func formatToFormatter(format string) []types.AccessLogFormatter {

	strArray := strings.Split(format, " ")

	// delete %
	for i := 0; i < len(strArray); i++ {
		strArray[i] = strArray[i][1 : len(strArray[i])-1]
	}

	// classify keys
	var reqInfoArray, reqHeaderArray, respHeaderArray []string
	for _, s := range strArray {
		if strings.HasPrefix(s, types.ReqHeaderPrefix) {
			reqHeaderArray = append(reqHeaderArray, s)

		} else if strings.HasPrefix(s, types.RespHeaderPrefix) {
			respHeaderArray = append(respHeaderArray, s)
		} else {
			reqInfoArray = append(reqInfoArray, s)
		}
	}

	// delete REQ.
	if reqHeaderArray != nil {
		for i := 0; i < len(reqHeaderArray); i++ {
			reqHeaderArray[i] = reqHeaderArray[i][len(types.ReqHeaderPrefix):]
		}
	}

	// delete RESP.
	if respHeaderArray != nil {
		for i := 0; i < len(respHeaderArray); i++ {
			respHeaderArray[i] = respHeaderArray[i][len(types.RespHeaderPrefix):]
		}
	}

	// set info function
	var infoFunc []func(info types.RequestInfo) string
	for _, key := range reqInfoArray {
		if vFunc, ok := RequestInfoFuncMap[key]; ok {
			infoFunc = append(infoFunc, vFunc)
		} else {
			DefaultLogger.Debugf("Invalid ReqInfo Format Keys: %s", key)
		}
	}

	return []types.AccessLogFormatter{
		&simpleRequestInfoFormatter{reqInfoFunc: infoFunc},
		&simpleReqHeadersFormatter{reqHeaderFormat: reqHeaderArray},
		&simpleRespHeadersFormatter{respHeaderFormat: respHeaderArray},
	}
}

// StartTimeGetter
// get request's arriving time
func StartTimeGetter(info types.RequestInfo) string {
	return info.StartTime().Format("2006/01/02 15:04:05.000")
}

// ReceivedDurationGetter
// get duration between request arriving and request resend to upstream
func ReceivedDurationGetter(info types.RequestInfo) string {
	return info.RequestReceivedDuration().String()
}

// ResponseReceivedDurationGetter
// get duration between request arriving and response sending
func ResponseReceivedDurationGetter(info types.RequestInfo) string {
	return info.ResponseReceivedDuration().String()
}

// RequestFinishedDurationGetter hets duration between request arriving and request finished
func RequestFinishedDurationGetter(info types.RequestInfo) string {
	return info.RequestFinishedDuration().String()
}

// BytesSentGetter
// get bytes sent
func BytesSentGetter(info types.RequestInfo) string {
	return strconv.FormatUint(info.BytesSent(), 10)
}

// BytesReceivedGetter
// get bytes received
func BytesReceivedGetter(info types.RequestInfo) string {
	return strconv.FormatUint(info.BytesReceived(), 10)
}

// get request's protocol type
func ProtocolGetter(info types.RequestInfo) string {
	return string(info.Protocol())
}

// ResponseCodeGetter
// get request's response code
func ResponseCodeGetter(info types.RequestInfo) string {
	return strconv.FormatUint(uint64(info.ResponseCode()), 10)
}

// DurationGetter
// get duration since request's starting time
func DurationGetter(info types.RequestInfo) string {
	return info.Duration().String()
}

// GetResponseFlagGetter
// get request's response flag
func GetResponseFlagGetter(info types.RequestInfo) string {
	return strconv.FormatBool(info.GetResponseFlag(0))
}

// UpstreamLocalAddressGetter
// get upstream's local address
func UpstreamLocalAddressGetter(info types.RequestInfo) string {
	if info.UpstreamLocalAddress() != nil {
		return info.UpstreamLocalAddress().String()
	}
	return ""
}

// DownstreamLocalAddressGetter
// get downstream's local address
func DownstreamLocalAddressGetter(info types.RequestInfo) string {
	if info.DownstreamLocalAddress() != nil {
		return info.DownstreamLocalAddress().String()
	}
	return ""
}

// DownstreamRemoteAddressGetter
// get upstream's remote address
func DownstreamRemoteAddressGetter(info types.RequestInfo) string {
	if info.DownstreamRemoteAddress() != nil {
		return info.DownstreamRemoteAddress().String()
	}
	return ""
}

// UpstreamHostSelectedGetter
// get upstream's selected host address
func UpstreamHostSelectedGetter(info types.RequestInfo) string {
	if info.UpstreamHost() != nil {
		return info.UpstreamHost().Hostname()
	}
	return ""
}
