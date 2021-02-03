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
	"time"

	v3 "github.com/SkyAPM/go2sky/reporter/grpc/language-agent"
)

type NoopSpan struct {
}

func (*NoopSpan) SetOperationName(string) {
}

func (*NoopSpan) GetOperationName() string {
	return ""
}

func (*NoopSpan) SetPeer(string) {
}

func (*NoopSpan) SetSpanLayer(v3.SpanLayer) {
}

func (*NoopSpan) SetComponent(int32) {
}

func (*NoopSpan) Tag(Tag, string) {
}

func (*NoopSpan) Log(time.Time, ...string) {
}

func (*NoopSpan) Error(time.Time, ...string) {
}

func (*NoopSpan) End() {
}

func (*NoopSpan) IsEntry() bool {
	return false
}

func (*NoopSpan) IsExit() bool {
	return false
}
