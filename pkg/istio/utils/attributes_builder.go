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

package utils

import (
	"time"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"

	google_protobuf2 "github.com/gogo/protobuf/types"

	"istio.io/api/mixer/v1"
)

type AttributesBuilder struct {
	attributes *v1.Attributes
}

func NewAttributesBuilder(attributes *v1.Attributes) *AttributesBuilder {
	return &AttributesBuilder{
		attributes:attributes,
	}
}

func (a *AttributesBuilder) HasAttribute(key string) bool {
	_, exist := a.attributes.Attributes[key]
	return exist
}

func (a *AttributesBuilder) AddBytes(key string, data []byte) {
	a.attributes.Attributes[key] = &v1.Attributes_AttributeValue{
		Value:&v1.Attributes_AttributeValue_BytesValue{
			BytesValue:data,
		},
	}
}

func (a *AttributesBuilder) AddInt64(key string, data int64) {
	a.attributes.Attributes[key] = &v1.Attributes_AttributeValue{
		Value:&v1.Attributes_AttributeValue_Int64Value{
			Int64Value:data,
		},
	}
}

func (a *AttributesBuilder) AddStringMap(key string, stringMap types.HeaderMap) {
	c, ok := stringMap.(protocol.CommonHeader)
	if !ok || len(c) == 0 {
		return
	}

	entries := &v1.Attributes_AttributeValue_StringMapValue{
		StringMapValue: &v1.Attributes_StringMap{
			Entries:make(map[string]string, 0),
		},
	}

	for k, v := range c {
		entries.StringMapValue.Entries[k] = v
	}

	a.attributes.Attributes[key] = &v1.Attributes_AttributeValue{
		Value:entries,
	}
}

func (a *AttributesBuilder) AddTimestamp(key string, t time.Time) {
	nano := t.UnixNano()

	timeStamp := &v1.Attributes_AttributeValue_TimestampValue{
		TimestampValue:&google_protobuf2.Timestamp{

		},
	}

	a.attributes.Attributes[key] = &v1.Attributes_AttributeValue{
		Value:timeStamp,
	}
}

func (a *AttributesBuilder) AddDuration(key string, value time.Duration) {
	/*
	a.attributes.Attributes[key] = &v1.Attributes_AttributeValue{
		Value:&v1.Attributes_AttributeValue_DurationValue{
			DurationValue:google_protobuf2.Timestamp{

			}
		},
	}
	*/
}

func (a *AttributesBuilder) AddString(key string, value string) {
	a.attributes.Attributes[key] = &v1.Attributes_AttributeValue{
		Value:&v1.Attributes_AttributeValue_StringValue{
			StringValue:value,
		},
	}
}