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

package stats

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	v1 "istio.io/api/mixer/v1"
	"mosn.io/api"
	"mosn.io/mosn/pkg/cel/attribute"
	"mosn.io/mosn/pkg/istio/utils"
	"mosn.io/mosn/pkg/protocol"
)

func ExtractAttributes(reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo, requestTotalSize uint64, now time.Time) attribute.Bag {
	return &extractAttributes{
		reqHeaders:       reqHeaders,
		respHeaders:      respHeaders,
		requestInfo:      requestInfo,
		requestTotalSize: requestTotalSize,
		now:              now,
		extracted:        map[string]interface{}{},
	}
}

type extractAttributes struct {
	reqHeaders       api.HeaderMap
	respHeaders      api.HeaderMap
	requestInfo      api.RequestInfo
	requestTotalSize uint64
	now              time.Time
	attributes       map[string]interface{}
	extracted        map[string]interface{}
}

func (e *extractAttributes) Get(name string) (interface{}, bool) {
	v, ok := e.extracted[name]
	if ok {
		if v == nil {
			return nil, false
		}
		return v, ok
	}

	switch name {
	case utils.KOriginIP:
		addr := e.requestInfo.DownstreamLocalAddress()
		if addr != nil {
			ip, _, ret := getIPPort(addr.String())
			if ret {
				v := []byte(ip)
				e.extracted[utils.KOriginIP] = v
				return v, true
			}
		}
		e.extracted[utils.KOriginIP] = nil
	case utils.KRequestTime:
		return e.requestInfo.StartTime(), true
	case utils.KContextProtocol:
		proto := "http"
		switch t := e.requestInfo.Protocol(); t {
		case protocol.HTTP1:
			proto = "http"
		case protocol.HTTP2:
			proto = "h2"
		default:
			if t != "" {
				proto = string(t)
			}
		}
		e.extracted[utils.KContextProtocol] = proto
		return proto, true
	case utils.KDestinationIP, utils.KDestinationPort:
		hostInfo := e.requestInfo.UpstreamHost()
		if hostInfo != nil {
			address := hostInfo.AddressString()
			ip, port, ret := getIPPort(address)
			if ret {
				if e.extracted[utils.KDestinationIP] == nil {
					e.extracted[utils.KDestinationIP] = []byte(ip)
				}
				if e.extracted[utils.KDestinationPort] == nil {
					e.extracted[utils.KDestinationPort] = int64(port)
				}
				return e.extracted[name], true
			}
		}
		e.extracted[name] = nil
	case utils.KResponseHeaders:
		return e.respHeaders, true
	case utils.KResponseTime:
		return e.now, true
	case utils.KRequestBodySize:
		return int64(e.requestInfo.BytesReceived()), true
	case utils.KResponseBodySize:
		return int64(e.requestInfo.BytesSent()), true
	case utils.KRequestTotalSize:
		return int64(e.requestTotalSize), true
	case utils.KResponseTotalSize:
		return int64(e.requestInfo.BytesSent() + e.respHeaders.ByteSize()), true
	case utils.KResponseDuration:
		return e.requestInfo.Duration(), true
	case utils.KResponseCode:
		return int64(e.requestInfo.ResponseCode()), true
	case utils.KRequestPath:
		return e.reqHeaders.Get(protocol.MosnHeaderPathKey)
	case utils.KRequestMethod:
		return e.reqHeaders.Get(protocol.MosnHeaderMethod)
	case utils.KRequestHost:
		return e.reqHeaders.Get(protocol.MosnHeaderHostKey)
	default:
		if e.attributes == nil {
			e.attributes = map[string]interface{}{}
			val, ret := e.reqHeaders.Get(utils.KIstioAttributeHeader)
			if ret {
				d, err := base64.StdEncoding.DecodeString(val)
				if err == nil {
					var attibutes v1.Attributes
					err = proto.Unmarshal(d, &attibutes)
					if err == nil {
						e.attributes = attributesToStringInterfaceMap(e.attributes, attibutes)
					}
				}
			}
		}
		v, ok = e.attributes[name]
		if ok {
			return v, ok
		}
	}
	return nil, false
}

// getIPPort return ip and port of address
func getIPPort(address string) (ip string, port int32, ret bool) {
	ret = false
	array := strings.Split(address, ":")
	if len(array) != 2 {
		return
	}
	p, err := strconv.Atoi(array[1])
	if err != nil {
		return
	}

	ip = array[0]
	port = int32(p)
	ret = true
	return
}

func attributesToStringInterfaceMap(out map[string]interface{}, attributes v1.Attributes) map[string]interface{} {
	if out == nil {
		out = map[string]interface{}{}
	}
	for key, val := range attributes.Attributes {
		var v interface{}
		switch t := val.Value.(type) {
		case *v1.Attributes_AttributeValue_StringValue:
			v = t.StringValue
		case *v1.Attributes_AttributeValue_Int64Value:
			v = t.Int64Value
		case *v1.Attributes_AttributeValue_DoubleValue:
			v = t.DoubleValue
		case *v1.Attributes_AttributeValue_BoolValue:
			v = t.BoolValue
		case *v1.Attributes_AttributeValue_BytesValue:
			v = t.BytesValue
		case *v1.Attributes_AttributeValue_TimestampValue:
			v = time.Unix(t.TimestampValue.Seconds, int64(t.TimestampValue.Nanos))
		case *v1.Attributes_AttributeValue_DurationValue:
			v = time.Duration(t.DurationValue.Seconds)*time.Second + time.Duration(t.DurationValue.Nanos)*time.Nanosecond
		case *v1.Attributes_AttributeValue_StringMapValue:
			v = protocol.CommonHeader(t.StringMapValue.Entries)
		}
		if v != nil {
			out[key] = v
		}
	}
	return out
}
