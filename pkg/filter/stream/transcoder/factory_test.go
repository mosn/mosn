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

package transcoder

import (
	"context"
	"mosn.io/api/extensions/transcoder"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
)

type tt struct {
	transcoder.Transcoder
}

type mockCallback struct {
	receiverPhase     api.ReceiverFilterPhase
	senderFilterPhase api.SenderFilterPhase
}

func (mock *mockCallback) AddStreamSenderFilter(filter api.StreamSenderFilter, p api.SenderFilterPhase) {
	mock.senderFilterPhase = p
}

func (mock *mockCallback) AddStreamReceiverFilter(filter api.StreamReceiverFilter, p api.ReceiverFilterPhase) {
	mock.receiverPhase = p
}

func (mock *mockCallback) AddStreamAccessLog(accessLog api.AccessLog) {
}

func TestCreateFilter(t *testing.T) {
	testcase := []struct {
		conf           map[string]interface{}
		expectReceiver api.ReceiverFilterPhase
	}{
		{
			conf: map[string]interface{}{
				"type":  "http2bolt_simple",
				"trans": map[string]interface{}{"receiver_phase": float64(api.AfterChooseHost)},
			},
			expectReceiver: api.AfterChooseHost,
		},
		{
			conf: map[string]interface{}{
				"type": "http2bolt_simple",
			},
			expectReceiver: api.AfterRoute,
		},
		{
			conf: map[string]interface{}{
				"type":  "http2bolt_simple",
				"trans": map[string]interface{}{"receiver_phase": float64(100)},
			},
			expectReceiver: api.AfterRoute,
		},
		{
			conf: map[string]interface{}{
				"type":  "http2bolt_simple",
				"trans": map[string]interface{}{"receiver_phase": float64(api.BeforeRoute)},
			},
			expectReceiver: api.BeforeRoute,
		},
		{
			conf: map[string]interface{}{
				"type":  "http2bolt",
				"trans": map[string]interface{}{"receiver_phase": float64(api.BeforeRoute)},
			},
			expectReceiver: api.BeforeRoute,
		},
	}
	MustRegister("http2bolt_simple", &tt{})
	for _, tcase := range testcase {
		ff, err := createFilterChainFactory(tcase.conf)
		assert.NoError(t, err)
		mock := &mockCallback{}
		ff.CreateFilterChain(context.Background(), mock)
		assert.Equal(t, tcase.expectReceiver, mock.receiverPhase)
	}
}
