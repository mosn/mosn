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

import "istio.io/api/mixer/v1"

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