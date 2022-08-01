//go:build wasmer
// +build wasmer

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

package v2

import (
	"context"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/mosn/pkg/mock"
	_ "mosn.io/mosn/pkg/stream/http"
	"mosn.io/mosn/pkg/wasm/abi/proxywasm020"
	_ "mosn.io/mosn/pkg/wasm/runtime/wasmer"
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
			"engine": "wasmer",
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

func TestFilterHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	f := &Filter{}
	assert.Nil(t, f.GetHttpRequestHeader())
	assert.Nil(t, f.GetHttpRequestBody())
	assert.Nil(t, f.GetHttpRequestTrailer())
	assert.Nil(t, f.GetHttpResponseHeader())
	assert.Nil(t, f.GetHttpResponseBody())
	assert.Nil(t, f.GetHttpResponseTrailer())

	reqHeader := mockHeaderMap(ctrl)
	reqBody := buffer.NewIoBufferString("request body")
	reqTrailer := mockHeaderMap(ctrl)

	receiverHandler := mock.NewMockStreamReceiverFilterHandler(ctrl)
	receiverHandler.EXPECT().GetRequestHeaders().AnyTimes().Return(reqHeader)
	receiverHandler.EXPECT().GetRequestData().AnyTimes().Return(reqBody)
	receiverHandler.EXPECT().GetRequestTrailers().AnyTimes().Return(reqTrailer)

	respHeader := mockHeaderMap(ctrl)
	respBody := buffer.NewIoBufferString("response body")
	respTrailer := mockHeaderMap(ctrl)

	senderHandler := mock.NewMockStreamSenderFilterHandler(ctrl)
	senderHandler.EXPECT().GetResponseHeaders().AnyTimes().Return(respHeader)
	senderHandler.EXPECT().GetResponseData().AnyTimes().Return(respBody)
	senderHandler.EXPECT().GetResponseTrailers().AnyTimes().Return(respTrailer)

	f.SetReceiveFilterHandler(receiverHandler)
	f.SetSenderFilterHandler(senderHandler)

	assert.Equal(t, f.GetHttpRequestHeader().(*proxywasm020.HeaderMapWrapper).HeaderMap, reqHeader)
	assert.Equal(t, f.GetHttpRequestBody().(*proxywasm020.IoBufferWrapper).IoBuffer, reqBody)
	assert.Equal(t, f.GetHttpRequestTrailer().(*proxywasm020.HeaderMapWrapper).HeaderMap, reqTrailer)
	assert.Equal(t, f.GetHttpResponseHeader().(*proxywasm020.HeaderMapWrapper).HeaderMap, respHeader)
	assert.Equal(t, f.GetHttpResponseBody().(*proxywasm020.IoBufferWrapper).IoBuffer, respBody)
	assert.Equal(t, f.GetHttpResponseTrailer().(*proxywasm020.HeaderMapWrapper).HeaderMap, respTrailer)
}
