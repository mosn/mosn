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
	"context"
	"testing"

	monkey "github.com/cch123/supermonkey"
	"github.com/golang/mock/gomock"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mock"
)

func TestStreamFilterFactory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer monkey.UnpatchAll()

	createFilterChainCount := 0

	monkey.Patch(createStreamFilterFactoryFromConfig, func(configs []v2.Filter) []api.StreamFilterChainFactory {
		sfcf := mock.NewMockStreamFilterChainFactory(ctrl)
		sfcf.EXPECT().CreateFilterChain(gomock.Any(), gomock.Any()).Do(func(context.Context, api.StreamFilterChainFactoryCallbacks) {
			createFilterChainCount++
		}).AnyTimes()
		return []api.StreamFilterChainFactory{sfcf, sfcf}
	})

	factory := NewStreamFilterFactory(nil)
	factory.CreateFilterChain(context.TODO(), nil)
	if createFilterChainCount != 2 {
		t.Errorf("createFilterChainCount=%v, want=2", createFilterChainCount)
	}
}
