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

	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/valyala/bytebufferpool"
)

// RequestInfoFuncMap is a map which key is the format-key, value is the func to get corresponding string value
var (
	RequestInfoFuncMap map[string]func(info types.RequestInfo) string
	// currently use fasthttp's bytebufferpool impl
	accessLogPool bytebufferpool.Pool
)

func init() {
	RequestInfoFuncMap = map[string]func(info types.RequestInfo) string{
		types.LogStartTime:                  StartTimeGetter,
		types.LogRequestReceivedDuration:    ReceivedDurationGetter,
		types.LogResponseReceivedDuration:   ResponseReceivedDurationGetter,
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
}

// types.AccessLog
type accesslog struct {
	output    string
	filter    types.AccessLogFilter
	formatter types.AccessLogFormatter
	logger    Logger
}

// NewAccessLog
func NewAccessLog(output string, filter types.AccessLogFilter,
	format string) (types.AccessLog, error) {
	var err error
	var logger Logger

	if logger, err = GetLoggerInstance(output, 0); err != nil {
		return nil, err
	}

	return &accesslog{
		output:    output,
		filter:    filter,
		formatter: NewAccessLogFormatter(format),
		logger:    logger,
	}, nil
}

func (l *accesslog) Log(reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) {
	if l.filter != nil {
		if !l.filter.Decide(reqHeaders, requestInfo) {
			return
		}
	}

	l.logger.Println(l.formatter.Format(reqHeaders, respHeaders, requestInfo))
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

func (f *accesslogformatter) Format(reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) string {
	var log string

	for _, formatter := range f.formatters {
		log += formatter.Format(reqHeaders, respHeaders, requestInfo)
	}

	//delete the final " "
	if len(log) > 0 {
		log = log[:len(log)-1]
	}

	return log
}

// types.AccessLogFormatter
type simpleRequestInfoFormatter struct {
	reqInfoFormat []string
}

// Format request info headers
func (f *simpleRequestInfoFormatter) Format(reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) string {
	// todo: map fieldName to field vale string
	if f.reqInfoFormat == nil {
		DefaultLogger.Debugf("No ReqInfo Format Keys Input")
		return ""
	}

	buffer := accessLogPool.Get()
	defer accessLogPool.Put(buffer)
	for _, key := range f.reqInfoFormat {

		if vFunc, ok := RequestInfoFuncMap[key]; ok {
			buffer.WriteString(vFunc(requestInfo))
			buffer.WriteString(" ")
		} else {
			DefaultLogger.Debugf("Invalid ReqInfo Format Keys: %s", key)
		}
	}

	return buffer.String()
}

// types.AccessLogFormatter
type simpleReqHeadersFormatter struct {
	reqHeaderFormat []string
}

// Format request headers format
func (f *simpleReqHeadersFormatter) Format(reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) string {
	if f.reqHeaderFormat == nil {
		DefaultLogger.Debugf("No ReqHeaders Format Keys Input")
		return ""
	}

	buffer := accessLogPool.Get()
	defer accessLogPool.Put(buffer)

	for _, key := range f.reqHeaderFormat {
		if v, ok := reqHeaders.Get(key); ok {
			buffer.WriteString(types.ReqHeaderPrefix)
			buffer.WriteString(v)
			buffer.WriteString(" ")
		} else {
			//DefaultLogger.Debugf("Invalid reqHeaders format keys when print access log: %s", key)
		}
	}

	return buffer.String()
}

// types.AccessLogFormatter
type simpleRespHeadersFormatter struct {
	respHeaderFormat []string
}

// Format response headers format
func (f *simpleRespHeadersFormatter) Format(reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) string {
	if f.respHeaderFormat == nil {
		DefaultLogger.Debugf("No RespHeaders Format Keys Input")
		return ""
	}

	buffer := accessLogPool.Get()
	defer accessLogPool.Put(buffer)
	for _, key := range f.respHeaderFormat {

		if v, ok := respHeaders.Get(key); ok {
			buffer.WriteString(types.RespHeaderPrefix)
			buffer.WriteString(v)
			buffer.WriteString(" ")
		} else {
			//DefaultLogger.Debugf("Invalid RespHeaders Format Keys:%s", key)
		}
	}

	return buffer.String()
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

	return []types.AccessLogFormatter{
		&simpleRequestInfoFormatter{reqInfoFormat: reqInfoArray},
		&simpleReqHeadersFormatter{reqHeaderFormat: reqHeaderArray},
		&simpleRespHeadersFormatter{respHeaderFormat: respHeaderArray},
	}
}

// StartTimeGetter
// get request's arriving time
func StartTimeGetter(info types.RequestInfo) string {
	return info.StartTime().Format("2006-01-02 15:04:05.999 +800")
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
	return "nil"
}

// DownstreamLocalAddressGetter
// get downstream's local address
func DownstreamLocalAddressGetter(info types.RequestInfo) string {
	if info.DownstreamLocalAddress() != nil {
		return info.DownstreamLocalAddress().String()
	}
	return "nil"
}

// DownstreamRemoteAddressGetter
// get upstream's remote address
func DownstreamRemoteAddressGetter(info types.RequestInfo) string {
	if info.DownstreamRemoteAddress() != nil {
		return info.DownstreamRemoteAddress().String()
	}
	return "nil"
}

// UpstreamHostSelectedGetter
// get upstream's selected host address
func UpstreamHostSelectedGetter(info types.RequestInfo) string {
	if info.UpstreamHost() != nil {
		return info.UpstreamHost().Hostname()
	}
	return "nil"
}
