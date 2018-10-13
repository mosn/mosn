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

package mixerclient

import (
	"time"

	"istio.io/api/mixer/v1"
)

type BatchCompressor interface {
	Add(attributes *v1.Attributes)
}

type AttributeCompressor struct {
 	globalDict *GlobalDictionary
}

type batchCompressor struct {
	globalDict *GlobalDictionary
	dict       *MessageDictionary
	report 		v1.ReportRequest
}

type MessageDictionary struct {
	globalDict *GlobalDictionary
	messageDict map[string]int32
	messageWords []string
}

// GlobalDictionary store global dictionary
type GlobalDictionary struct {
	globalDict map[string]int32
	topIndex int32
}

func NewAttributeCompressor() *AttributeCompressor {
	return &AttributeCompressor{
		globalDict:NewGlobalDictionary(),
	}
}

func (a *AttributeCompressor) CreateBatchCompressor() BatchCompressor {
	return NewBatchCompressor(a.globalDict)
}

func NewBatchCompressor(globalDict *GlobalDictionary) BatchCompressor {
	b := &batchCompressor{
		globalDict:globalDict,
		dict:NewMessageDictionary(globalDict),
	}

	b.report.Attributes = make([]v1.CompressedAttributes, 0)
	b.report.Attributes = append(b.report.Attributes, v1.CompressedAttributes{
		Words:make([]string, 0),
		Strings:make(map[int32]int32, 0),
		Int64S:make(map[int32]int64, 0),
		Doubles:make(map[int32]float64, 0),
		Bools:make(map[int32]bool, 0),
		Timestamps:make(map[int32]time.Time, 0),
		Durations:make(map[int32]time.Duration, 0),
		Bytes:make(map[int32][]byte, 0),
		StringMaps:make(map[int32]v1.StringMap, 0),
	})

	return b
}

func (b *batchCompressor) Add(attributes *v1.Attributes) {
	CompressByDict(attributes, b.dict, &b.report.Attributes[0])
}

func NewMessageDictionary(globalDict *GlobalDictionary) *MessageDictionary {
	return &MessageDictionary{
		globalDict:globalDict,
		messageDict:make(map[string]int32, 0),
		messageWords:make([]string, 0),
	}
}

func (m *MessageDictionary) GetIndex(key string) int32 {
	index, exist := m.globalDict.GetIndex(key)
	if exist {
		return index
	}

	index, exist = m.messageDict[key]
	if exist {
		return index
	}

	index = int32(len(m.messageWords))
	m.messageWords = append(m.messageWords, key)
	m.messageDict[key] = int32(index)

	return index
}

func NewGlobalDictionary() *GlobalDictionary {
	g := &GlobalDictionary{
		globalDict:make(map[string]int32, 0),
	}

	for i, v := range KGlobalWords {
		g.globalDict[v] = int32(i)
	}

	g.topIndex = int32(len(g.globalDict))

	return g
}

func (g *GlobalDictionary) GetIndex(key string) (index int32, exist bool) {
	index, exist = g.globalDict[key]
	if exist && index < g.topIndex {
		return
	}
	exist = false
	return
}

func CompressByDict(attributes *v1.Attributes, dict *MessageDictionary, pb *v1.CompressedAttributes) {
	for k,v := range attributes.Attributes {
		index := dict.GetIndex(k)
		value := v.Value

		switch val := value.(type) {
		case *v1.Attributes_AttributeValue_StringValue:
			pb.Strings[index] = dict.GetIndex(val.StringValue)
		}

		bs, ok := value.(*v1.Attributes_AttributeValue_BytesValue)
		if ok {
			pb.Bytes[index] = bs.BytesValue
			continue
		}

		i64, ok := value.(*v1.Attributes_AttributeValue_Int64Value)
		if ok {
			pb.Int64S[index] = i64.Int64Value
			continue
		}

		d, ok := value.(*v1.Attributes_AttributeValue_DoubleValue)
		if ok {
			pb.Doubles[index] = d.DoubleValue
			continue
		}

		b, ok := value.(*v1.Attributes_AttributeValue_BoolValue)
		if ok {
			pb.Bools[index] = b.BoolValue
			continue
		}

		/*
		_, ok := value.(*v1.Attributes_AttributeValue_TimestampValue)
		if ok {
			//pb.Timestamps[index] = ts.TimestampValue.
			continue
		}

		dv, ok := value.(*v1.Attributes_AttributeValue_DurationValue)
		if ok {
			pb.Durations[index] = dv.DurationValue
			continue
		}
		*/
		sm, ok := value.(*v1.Attributes_AttributeValue_StringMapValue)
		if ok {
			pb.StringMaps[index] = CreateStringMap(sm.StringMapValue, dict)
			continue
		}
	}
}

func CreateStringMap(sm *v1.Attributes_StringMap, dict *MessageDictionary) v1.StringMap {
	var compressedMap v1.StringMap
	compressedMap.Entries = make(map[int32]int32, 0)
	entries := compressedMap.Entries

	for k, v := range(sm.Entries) {
		entries[dict.GetIndex(k)] = dict.GetIndex(v)
	}

	return compressedMap
}