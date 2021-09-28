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

package streamfilter

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/mosn/pkg/mock"
)

func TestStreamFilterChainPool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chain := GetDefaultStreamFilterChain()

	receiverFilter := mock.NewMockStreamReceiverFilter(ctrl)
	chain.AddStreamReceiverFilter(receiverFilter, api.BeforeRoute)

	senderFilter := mock.NewMockStreamSenderFilter(ctrl)
	chain.AddStreamSenderFilter(senderFilter, api.BeforeSend)

	PutStreamFilterChain(chain)

	chain2 := GetDefaultStreamFilterChain()

	assert.Equal(t, chain, chain2)

	assert.Equal(t, len(chain2.senderFilters), 0)
	assert.Equal(t, len(chain2.senderFiltersPhase), 0)
	assert.Equal(t, chain2.senderFiltersIndex, 0)
	assert.Equal(t, len(chain2.receiverFilters), 0)
	assert.Equal(t, len(chain2.receiverFiltersPhase), 0)
	assert.Equal(t, chain2.receiverFiltersIndex, 0)
}

func TestStreamFilterChainSetFilterHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chain := GetDefaultStreamFilterChain()

	setHandlerCount := 0

	for i := 0; i < 5; i++ {
		receiverFilter := mock.NewMockStreamReceiverFilter(ctrl)
		receiverFilter.EXPECT().SetReceiveFilterHandler(gomock.Any()).AnyTimes().Do(func(api.StreamReceiverFilterHandler) {
			setHandlerCount++
		})
		chain.AddStreamReceiverFilter(receiverFilter, api.BeforeRoute)

		senderFilter := mock.NewMockStreamSenderFilter(ctrl)
		senderFilter.EXPECT().SetSenderFilterHandler(gomock.Any()).AnyTimes().Do(func(api.StreamSenderFilterHandler) {
			setHandlerCount++
		})
		chain.AddStreamSenderFilter(senderFilter, api.BeforeSend)
	}

	chain.SetReceiveFilterHandler(nil)
	rFilterHandler := mock.NewMockStreamReceiverFilterHandler(ctrl)
	chain.SetReceiveFilterHandler(rFilterHandler)

	chain.SetSenderFilterHandler(nil)
	sFilterHandler := mock.NewMockStreamSenderFilterHandler(ctrl)
	chain.SetSenderFilterHandler(sFilterHandler)

	assert.Equal(t, setHandlerCount, 10)
}
