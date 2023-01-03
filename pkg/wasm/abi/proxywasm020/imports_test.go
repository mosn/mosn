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

package proxywasm020

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/mock"
	"mosn.io/pkg/buffer"
	"mosn.io/proxy-wasm-go-host/proxywasm/common"
	proxywasm "mosn.io/proxy-wasm-go-host/proxywasm/v2"
)

func TestImportsHandler(t *testing.T) {
	d := &DefaultImportsHandler{}
	assert.Equal(t, d.Log(proxywasm.LogLevelError, "msg"), proxywasm.ResultOk)
	assert.Equal(t, d.Log(proxywasm.LogLevelWarning, "msg"), proxywasm.ResultOk)
	assert.Equal(t, d.Log(proxywasm.LogLevelInfo, "msg"), proxywasm.ResultOk)
	assert.Equal(t, d.Log(proxywasm.LogLevelDebug, "msg"), proxywasm.ResultOk)
	assert.Equal(t, d.Log(proxywasm.LogLevelTrace, "msg"), proxywasm.ResultOk)
}

func TestImportsHandlerHttpCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := http.Server{Addr: ":22165"}
	defer server.Close()

	go func() {
		http.HandleFunc("/hihi", func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte("response from external server"))
		})
		server.ListenAndServe()
	}()
	time.Sleep(time.Second)

	instance := mock.NewMockWasmInstance(ctrl)
	abiContext := ABIContextFactory(instance)
	instance.EXPECT().GetData().AnyTimes().Return(abiContext)
	instance.EXPECT().Lock(gomock.Any()).AnyTimes().Return()
	instance.EXPECT().Unlock().AnyTimes().Return()
	instance.EXPECT().GetExportsFunc(gomock.Any()).AnyTimes().Return(nil, errors.New("func not exists"))

	d := abiContext.GetABIImports().(*DefaultImportsHandler)

	reqHeader := common.CommonHeader(map[string]string{
		"reqHeader1": "reqValue1",
		"reqHeader2": "reqValue2",
	})
	reqBody := &IoBufferWrapper{buffer.NewIoBufferBytes([]byte("req body"))}
	_, res := d.DispatchHttpCall("http://127.0.0.1:22165/hihi", reqHeader, reqBody, nil, 5000)
	assert.NotNil(t, res, proxywasm.ResultOk)
	assert.NotNil(t, d.hc)
	assert.True(t, d.hc.reqOnFly)

	d.Wait()

	assert.NotNil(t, d.GetHttpCallResponseHeaders())
	assert.NotNil(t, d.GetHttpCalloutResponseBody())
}
