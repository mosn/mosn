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

package types

//    The bunch of interfaces are used to print the access log in format designed by users.
//    Access log format consists of three parts, which are "RequestInfoFormat", "RequestHeaderFormat"
//    and "ResponseHeaderFormat", also you can get details by reading "AccessLogDetails.md".

// AccessLog is a log object that used to log the access info.
type AccessLog interface {
	// Log write the access info.
	// The "reqHeaders" contains the request header's information, "respHeader" contains the response header's information
	// and "requestInfo" contains some request information
	Log(reqHeaders HeaderMap, respHeaders HeaderMap, requestInfo RequestInfo)
}

// AccessLogFilter is a filter of access log to do some filters to access log info
type AccessLogFilter interface {
	// Decide can make a decision about how to filter the request headers and requestInfo
	Decide(reqHeaders HeaderMap, requestInfo RequestInfo) bool
}

// AccessLogFormatter is a object that format the request info to string
type AccessLogFormatter interface {
	// Format makes the request headers, response headers and request info to string for printing according to log formatter
	Format(reqHeaders HeaderMap, respHeaders HeaderMap, requestInfo RequestInfo) string
}

// The identification of a request info's content
const (
	LogStartTime                  string = "StartTime"
	LogRequestReceivedDuration    string = "RequestReceivedDuration"
	LogResponseReceivedDuration   string = "ResponseReceivedDuration"
	LogBytesSent                  string = "BytesSent"
	LogBytesReceived              string = "BytesReceived"
	LogProtocol                   string = "Protocol"
	LogResponseCode               string = "ResponseCode"
	LogDuration                   string = "Duration"
	LogResponseFlag               string = "ResponseFlag"
	LogUpstreamLocalAddress       string = "UpstreamLocalAddress"
	LogDownstreamLocalAddress     string = "DownstreamLocalAddress"
	LogDownstreamRemoteAddress    string = "DownstreamRemoteAddress"
	LogUpstreamHostSelectedGetter string = "UpstreamHostSelected"
)

const (
	// ReqHeaderPrefix is the prefix of request header's formatter
	ReqHeaderPrefix string = "REQ."
	// RespHeaderPrefix is the prefix of response header's formatter
	RespHeaderPrefix string = "RESP."
)

const (
	// DefaultAccessLogFormat is the default access log format.
	// For more details please read "AccessLogDetails.md"
	DefaultAccessLogFormat = "%StartTime% %RequestReceivedDuration% %ResponseReceivedDuration% %BytesSent%" + " " +
		"%BytesReceived% %Protocol% %ResponseCode% %Duration% %ResponseFlag% %ResponseCode% %UpstreamLocalAddress%" + " " +
		"%DownstreamLocalAddress% %DownstreamRemoteAddress% %UpstreamHostSelected%"
)
