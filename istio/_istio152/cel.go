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

package istio152

import (
	"encoding/base64"
	"time"

	"github.com/gogo/protobuf/proto"
	"istio.io/api/mixer/v1"
	"mosn.io/mosn/pkg/cel/extract"
	"mosn.io/mosn/pkg/protocol"
)

func init() {
	extract.RegisterAttributeGenerator(GenerateMixerV1Attribute)
}

func GenerateMixerV1Attribute(s string) map[string]interface{} {
	d, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil
	}
	var attributes v1.Attributes
	if err := proto.Unmarshal(d, &attributes); err != nil {
		return nil
	}
	// attributes to map
	return attributesToStringInterfaceMap(attributes)
}

func attributesToStringInterfaceMap(attributes v1.Attributes) map[string]interface{} {
	out := map[string]interface{}{}
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
