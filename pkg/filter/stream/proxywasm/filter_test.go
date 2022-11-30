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

package proxywasm

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/mosn/pkg/mock"
	_ "mosn.io/mosn/pkg/stream/http"
	_ "mosn.io/mosn/pkg/wasm/runtime/wazero"
	"mosn.io/pkg/buffer"
)

func mockHeaderMap(ctrl *gomock.Controller) api.HeaderMap {
	var m = map[string]string{
		"requestHeaderKey1": "requestHeaderValue1",
		"requestHeaderKey2": "requestHeaderValue2",
		"requestHeaderKey3": "requestHeaderValue3",
	}

	h := mock.NewMockHeaderMap(ctrl)

	h.EXPECT().Get(gomock.Any()).AnyTimes().DoAndReturn(func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	})
	h.EXPECT().Del(gomock.Any()).AnyTimes().DoAndReturn(func(key string) { delete(m, key) })
	h.EXPECT().Add(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(key string, val string) { m[key] = val })
	h.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(key string, val string) { m[key] = val })
	h.EXPECT().Range(gomock.Any()).AnyTimes().Do(func(f func(key, value string) bool) {
		for k, v := range m {
			if !f(k, v) {
				break
			}
		}
	})
	h.EXPECT().ByteSize().AnyTimes().DoAndReturn(func() uint64 {
		var size uint64
		for k, v := range m {
			size += uint64(len(k) + len(v))
		}
		return size
	})

	return h
}

func testProxyWasmStreamFilterCommon(t *testing.T, wasmPath string) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configMap := map[string]interface{}{
		"instance_num": 2,
		"vm_config": map[string]interface{}{
			"engine": "wazero",
			"path":   wasmPath,
		},
		"user_config1": "user_value1",
		"user_config2": "user_value2",
	}

	factory, err := createProxyWasmFilterFactory(configMap)
	if err != nil || factory == nil {
		t.Errorf("fail to create filter factory")
		return
	}

	wg := sync.WaitGroup{}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			var rFilter api.StreamReceiverFilter
			var sFilter api.StreamSenderFilter

			cb := mock.NewMockStreamFilterChainFactoryCallbacks(ctrl)
			cb.EXPECT().AddStreamReceiverFilter(gomock.Any(), gomock.Any()).Do(func(receiverFilter api.StreamReceiverFilter, p api.ReceiverFilterPhase) {
				assert.Equal(t, p, api.BeforeRoute, "add receiver filter at wrong phase")
				rFilter = receiverFilter
			}).AnyTimes()
			cb.EXPECT().AddStreamSenderFilter(gomock.Any(), gomock.Any()).Do(func(senderFilter api.StreamSenderFilter, p api.SenderFilterPhase) {
				assert.Equal(t, p, api.BeforeSend, "add sender filter at wrong phase")
				sFilter = senderFilter
			}).AnyTimes()

			factory.CreateFilterChain(context.TODO(), cb)

			reqHeaderMap := mockHeaderMap(ctrl)
			reqBody := buffer.NewIoBufferString("request body")
			reqTrailer := mockHeaderMap(ctrl)

			receiverHandler := mock.NewMockStreamReceiverFilterHandler(ctrl)
			receiverHandler.EXPECT().GetRequestHeaders().AnyTimes().Return(reqHeaderMap)
			receiverHandler.EXPECT().GetRequestData().AnyTimes().Return(reqBody)
			receiverHandler.EXPECT().GetRequestTrailers().AnyTimes().Return(reqTrailer)

			respHeader := mockHeaderMap(ctrl)
			respBody := buffer.NewIoBufferString("response body")
			respTrailer := mockHeaderMap(ctrl)

			senderHandler := mock.NewMockStreamSenderFilterHandler(ctrl)
			senderHandler.EXPECT().GetResponseHeaders().AnyTimes().Return(respHeader)
			senderHandler.EXPECT().GetResponseData().AnyTimes().Return(respBody)
			senderHandler.EXPECT().GetResponseTrailers().AnyTimes().Return(respTrailer)

			// for coverage
			rFilter.SetReceiveFilterHandler(receiverHandler)
			sFilter.SetSenderFilterHandler(senderHandler)

			rFilter.OnReceive(context.TODO(), reqHeaderMap, reqBody, reqTrailer)
			sFilter.Append(context.TODO(), respHeader, respBody, respTrailer)

			rFilter.OnDestroy()
			sFilter.OnDestroy()
		}()
	}

	wg.Wait()
}

func TestProxyWasmStreamFilterGo(t *testing.T) {
	testProxyWasmStreamFilterCommon(t, "./data/test-go.wasm")
}

func TestProxyWasmStreamFilterC(t *testing.T) {
	testProxyWasmStreamFilterCommon(t, "./data/test-c.wasm")
}

func TestProxyWasmStreamFilterHttpCallout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := http.Server{Addr: ":22164"}
	defer server.Close()

	go func() {
		http.HandleFunc("/haha", func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte("response from external server"))
		})
		server.ListenAndServe()
	}()

	configMap := map[string]interface{}{
		"instance_num": 1,
		"vm_config": map[string]interface{}{
			"engine": "wazero",
			"path":   "./data/httpCall.wasm",
		},
	}

	factory, err := createProxyWasmFilterFactory(configMap)
	if err != nil || factory == nil {
		t.Errorf("fail to create filter factory")
		return
	}

	var rFilter api.StreamReceiverFilter

	cb := mock.NewMockStreamFilterChainFactoryCallbacks(ctrl)
	cb.EXPECT().AddStreamReceiverFilter(gomock.Any(), gomock.Any()).Do(func(receiverFilter api.StreamReceiverFilter, p api.ReceiverFilterPhase) {
		assert.Equal(t, p, api.BeforeRoute, "add receiver filter at wrong phase")
		rFilter = receiverFilter
	}).AnyTimes()
	cb.EXPECT().AddStreamSenderFilter(gomock.Any(), gomock.Any()).AnyTimes().Return()

	factory.CreateFilterChain(context.TODO(), cb)

	reqHeaderMap := mockHeaderMap(ctrl)
	reqBody := buffer.NewIoBufferString("request body")

	receiverHandler := mock.NewMockStreamReceiverFilterHandler(ctrl)
	receiverHandler.EXPECT().GetRequestHeaders().AnyTimes().Return(reqHeaderMap)
	receiverHandler.EXPECT().GetRequestData().AnyTimes().Return(reqBody)

	rFilter.SetReceiveFilterHandler(receiverHandler)
	rFilter.OnReceive(context.TODO(), reqHeaderMap, reqBody, nil)
	rFilter.OnDestroy()
}
