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

// BatchCompressor
type BatchCompressor interface {
	Add(attributes *v1.Attributes)
	Finish() *v1.ReportRequest
	Size() int
	Clear()
}

type AttributeCompressor struct {
 	globalDict *GlobalDictionary
}

type batchCompressor struct {
	globalDict *GlobalDictionary
	dict       *MessageDictionary
	report 			v1.ReportRequest
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
	b.report.DefaultWords = make([]string, 0)

	return b
}

func (b *batchCompressor) Add(attributes *v1.Attributes) {
	CompressByDict(attributes, b.dict, &b.report.Attributes[0])
}

func (b *batchCompressor) Finish() *v1.ReportRequest {
	words := b.dict.GetWords()
	for _, word := range words {
		b.report.DefaultWords = append(b.report.DefaultWords, word)
	}
	b.report.GlobalWordCount = uint32(len(b.globalDict.globalDict))

	return &b.report
}

func (b *batchCompressor) Size() int {
	return b.report.Size()
}

func (b *batchCompressor) Clear() {
	b.report.Reset()
	b.dict.Clear()
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

func (m *MessageDictionary) GetWords() []string {
	return m.messageWords
}

func (m *MessageDictionary) Clear() {
	m.messageWords = make([]string, 0)
	m.messageDict = make(map[string]int32, 0)
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
		case *v1.Attributes_AttributeValue_BytesValue:
			pb.Bytes[index] = val.BytesValue
		case *v1.Attributes_AttributeValue_Int64Value:
			pb.Int64S[index] = val.Int64Value
		case *v1.Attributes_AttributeValue_DoubleValue:
			pb.Doubles[index] = val.DoubleValue
		case *v1.Attributes_AttributeValue_BoolValue:
			pb.Bools[index] = val.BoolValue
		case *v1.Attributes_AttributeValue_TimestampValue:
			pb.Timestamps[index] = time.Unix(val.TimestampValue.Seconds, int64(val.TimestampValue.Nanos))
		case *v1.Attributes_AttributeValue_DurationValue:
			pb.Durations[index] = time.Duration(val.DurationValue.Seconds * int64(time.Second) + int64(val.DurationValue.Nanos))
		case *v1.Attributes_AttributeValue_StringMapValue:
			pb.StringMaps[index] = CreateStringMap(val.StringMapValue, dict)
		default:
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