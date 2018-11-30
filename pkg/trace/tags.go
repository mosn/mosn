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

package trace

const (
	TRACE_ID               string = "traceId"
	SPAN_ID                string = "spanId"
	PARENT_SPAN_ID         string = "parentSpanId"
	SERVICE_NAME           string = "serviceName"
	METHOD_NAME            string = "methodName"
	PROTOCOL               string = "protocol"
	RESULT_STATUS          string = "resultStatus"
	REQUEST_SIZE           string = "requestSize"
	RESPONSE_SIZE          string = "responseSize"
	UPSTREAM_HOST_ADDRESS  string = "upstreamHostAddress"
	DOWNSTEAM_HOST_ADDRESS string = "downstreamHostAddress"
	APP_NAME               string = "appName"
	SPAN_TYPE              string = "spanType"
	BAGGAGE_DATA           string = "baggageData"
	REQUEST_URL string = "requestURL"
)
