// Licensed to SkyAPM org under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. SkyAPM org licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package go2sky

import (
	"math"
	"time"

	"github.com/SkyAPM/go2sky/internal/tool"
	"github.com/SkyAPM/go2sky/propagation"
	"github.com/SkyAPM/go2sky/reporter/grpc/common"
	v3 "github.com/SkyAPM/go2sky/reporter/grpc/language-agent"
)

// SpanType is used to identify entry, exit and local
type SpanType int32

const (
	// SpanTypeEntry is a entry span, eg http server
	SpanTypeEntry SpanType = 0
	// SpanTypeExit is a exit span, eg http client
	SpanTypeExit SpanType = 1
	// SpanTypeLocal is a local span, eg local method invoke
	SpanTypeLocal SpanType = 2
)

// Span interface as common span specification
type Span interface {
	SetOperationName(string)
	GetOperationName() string
	SetPeer(string)
	SetSpanLayer(v3.SpanLayer)
	SetComponent(int32)
	Tag(Tag, string)
	Log(time.Time, ...string)
	Error(time.Time, ...string)
	End()
	IsEntry() bool
	IsExit() bool
}

func newLocalSpan(t *Tracer) *defaultSpan {
	return &defaultSpan{
		tracer:    t,
		StartTime: time.Now(),
		SpanType:  SpanTypeLocal,
	}
}

type defaultSpan struct {
	Refs          []*propagation.SpanContext
	tracer        *Tracer
	StartTime     time.Time
	EndTime       time.Time
	OperationName string
	Peer          string
	Layer         v3.SpanLayer
	ComponentID   int32
	Tags          []*common.KeyStringValuePair
	Logs          []*v3.Log
	IsError       bool
	SpanType      SpanType
}

// For Span
func (ds *defaultSpan) SetOperationName(name string) {
	ds.OperationName = name
}

func (ds *defaultSpan) GetOperationName() string {
	return ds.OperationName
}

func (ds *defaultSpan) SetPeer(peer string) {
	ds.Peer = peer
}

func (ds *defaultSpan) SetSpanLayer(layer v3.SpanLayer) {
	ds.Layer = layer
}

func (ds *defaultSpan) SetComponent(componentID int32) {
	ds.ComponentID = componentID
}

func (ds *defaultSpan) Tag(key Tag, value string) {
	ds.Tags = append(ds.Tags, &common.KeyStringValuePair{Key: string(key), Value: value})
}

func (ds *defaultSpan) Log(time time.Time, ll ...string) {
	data := make([]*common.KeyStringValuePair, 0, int32(math.Ceil(float64(len(ll))/2.0)))
	var kvp *common.KeyStringValuePair
	for i, l := range ll {
		if i%2 == 0 {
			kvp = &common.KeyStringValuePair{}
			data = append(data, kvp)
			kvp.Key = l
		} else {
			if kvp != nil {
				kvp.Value = l
			}
		}
	}
	ds.Logs = append(ds.Logs, &v3.Log{Time: tool.Millisecond(time), Data: data})
}

func (ds *defaultSpan) Error(time time.Time, ll ...string) {
	ds.IsError = true
	ds.Log(time, ll...)
}

func (ds *defaultSpan) End() {
	ds.EndTime = time.Now()
}

func (ds *defaultSpan) IsEntry() bool {
	return ds.SpanType == SpanTypeEntry
}

func (ds *defaultSpan) IsExit() bool {
	return ds.SpanType == SpanTypeExit
}

// SpanOption allows for functional options to adjust behaviour
// of a Span to be created by CreateLocalSpan
type SpanOption func(s *defaultSpan)

// Tag are supported by sky-walking engine.
// As default, all Tags will be stored, but these ones have
// particular meanings.
type Tag string

const (
	TagURL             Tag = "url"
	TagStatusCode      Tag = "status_code"
	TagHTTPMethod      Tag = "http.method"
	TagDBType          Tag = "db.type"
	TagDBInstance      Tag = "db.instance"
	TagDBStatement     Tag = "db.statement"
	TagDBBindVariables Tag = "db.bind_vars"
	TagMQQueue         Tag = "mq.queue"
	TagMQBroker        Tag = "mq.broker"
	TagMQTopic         Tag = "mq.topic"
)

const (
	ComponentIDHttpServer int32 = 49
)
