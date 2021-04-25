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

package proxywasm010

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"
	"mosn.io/proxy-wasm-go-host/proxywasm/common"
	proxywasm "mosn.io/proxy-wasm-go-host/proxywasm/v1"
)

type DefaultImportsHandler struct {
	proxywasm.DefaultImportsHandler
	Instance       common.WasmInstance
	hc             *httpCallout
	DirectResponse bool
}

// override
func (d *DefaultImportsHandler) Log(level proxywasm.LogLevel, msg string) proxywasm.WasmResult {
	logFunc := log.DefaultLogger.Infof

	switch level {
	case proxywasm.LogLevelTrace:
		logFunc = log.DefaultLogger.Tracef
	case proxywasm.LogLevelDebug:
		logFunc = log.DefaultLogger.Debugf
	case proxywasm.LogLevelInfo:
		logFunc = log.DefaultLogger.Infof
	case proxywasm.LogLevelWarn:
		logFunc = log.DefaultLogger.Warnf
	case proxywasm.LogLevelError:
		logFunc = log.DefaultLogger.Errorf
	case proxywasm.LogLevelCritical:
		logFunc = log.DefaultLogger.Errorf
	}

	logFunc(msg)

	return proxywasm.WasmResultOk
}

var httpCalloutID int32

type httpCallout struct {
	id         int32
	d          *DefaultImportsHandler
	instance   common.WasmInstance
	abiContext *ABIContext

	urlString string
	client    *http.Client
	req       *http.Request

	respChan    chan *http.Response
	calloutErr  error
	respHeader  api.HeaderMap
	respBody    buffer.IoBuffer
	respTrailer api.HeaderMap

	reqOnFly bool
}

func (hc *httpCallout) reset() {
	if hc == nil {
		return
	}

	hc.id = 0
	hc.reqOnFly = false
	hc.d, hc.instance, hc.abiContext = nil, nil, nil
	hc.urlString, hc.client, hc.req = "", nil, nil
	hc.respHeader, hc.respBody, hc.respTrailer = nil, nil, nil
	hc.calloutErr = nil
}

var httpCalloutPool = sync.Pool{
	New: func() interface{} {
		return &httpCallout{
			respChan: make(chan *http.Response),
		}
	},
}

func GetHttpCallout() *httpCallout {
	return httpCalloutPool.Get().(*httpCallout)
}

func PutHttpCallout(hc *httpCallout) {
	hc.reset()
	httpCalloutPool.Put(hc)
}

// override
func (d *DefaultImportsHandler) HttpCall(reqURL string, header common.HeaderMap, body common.IoBuffer,
	trailer common.HeaderMap, timeoutMilliseconds int32) (int32, proxywasm.WasmResult) {
	u, err := url.Parse(reqURL)
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm010][imports] HttpCall fail to parse url, err: %v, reqURL: %v", err, reqURL)
		return 0, proxywasm.WasmResultBadArgument
	}

	calloutID := atomic.AddInt32(&httpCalloutID, 1)

	if d.hc == nil {
		d.hc = GetHttpCallout()
	} else {
		d.hc.reset()
	}
	d.hc.id = calloutID
	d.hc.d = d
	d.hc.instance = d.Instance
	d.hc.abiContext = d.Instance.GetData().(*ABIContext)
	d.hc.urlString = reqURL

	d.hc.client = &http.Client{Timeout: time.Millisecond * time.Duration(timeoutMilliseconds)}

	// prepare http req
	d.hc.req, err = buildHttpReq(u.String(), header, body, trailer)
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm010][imports] HttpCall fail to create http req, err: %v, reqURL: %v", err, reqURL)
		return 0, proxywasm.WasmResultInternalFailure
	}

	d.hc.reqOnFly = true

	// launch http call asynchronously
	// should start a new goroutine here because the calloutID must be returned to wasm module immediately.
	utils.GoWithRecover(func() {
		resp, err := d.hc.client.Do(d.hc.req)
		if err != nil {
			d.hc.calloutErr = err
		}
		d.hc.respChan <- resp
	}, nil)

	return calloutID, proxywasm.WasmResultOk
}

// override
func (d *DefaultImportsHandler) Wait() proxywasm.Action {
	if d.hc == nil || !d.hc.reqOnFly {
		if d.DirectResponse {
			d.DirectResponse = false
			return proxywasm.ActionPause
		}
		return proxywasm.ActionContinue
	}

	// release the instance lock and wait for http resp
	d.Instance.Unlock()
	resp := <-d.hc.respChan
	d.Instance.Lock(d.hc.abiContext)

	d.hc.reqOnFly = false
	defer func() {
		if d.hc != nil {
			PutHttpCallout(d.hc)
			d.hc = nil
		}
	}()

	rootContextID := d.hc.abiContext.Imports.GetRootContextID()
	exports := d.hc.abiContext.GetExports()

	if d.hc.calloutErr != nil || resp == nil {
		log.DefaultLogger.Errorf("[proxywasm010][imports] HttpCall id: %v fail to do http req, err: %v, reqURL: %v",
			d.hc.id, d.hc.calloutErr, d.hc.urlString)

		// should call proxy_on_http_call_response to prevent memory leak
		_ = exports.ProxyOnHttpCallResponse(rootContextID, d.hc.id, 0, 0, 0)

		// If 'DispatchHttpCall' got called again in 'ProxyOnHttpCallResponse', we should keep waiting again.
		return d.Wait()
	}

	respBodyLen := 0
	d.hc.respHeader, d.hc.respBody, respBodyLen, d.hc.respTrailer = parseHttpResp(resp)

	err := exports.ProxyOnHttpCallResponse(rootContextID, d.hc.id, int32(len(resp.Header)), int32(respBodyLen), 0)
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm010][imports] httpCall id: %v fail to call ProxyOnHttpCallResponse, err: %v", d.hc.id, err)
	}

	// If the DispatchHttpCall got invoked again in 'ProxyOnHttpCallResponse', we should keep waiting again.
	return d.Wait()
}

func buildHttpReq(url string, header common.HeaderMap, body common.IoBuffer,
	trailer common.HeaderMap) (*http.Request, error) {
	method := http.MethodGet
	if header != nil {
		if m, ok := header.Get("Method"); ok {
			method = strings.ToUpper(m)
		}
	}

	req, err := http.NewRequest(method, url, buffer.NewIoBufferBytes(body.Bytes()))
	if err != nil {
		return nil, err
	}

	if header != nil {
		header.Range(func(key, value string) bool {
			req.Header.Add(key, value)
			return true
		})
	}

	if trailer != nil {
		trailer.Range(func(key, value string) bool {
			req.Trailer.Add(key, value)
			return true
		})
	}

	return req, nil
}

func parseHttpResp(resp *http.Response) (respHeader api.HeaderMap, respBody buffer.IoBuffer, respBodyLen int, respTrailer api.HeaderMap) {
	if resp == nil {
		return nil, nil, 0, nil
	}

	// header
	if len(resp.Header) > 0 {
		respHeader = protocol.CommonHeader{}
		for key, _ := range resp.Header {
			respHeader.Set(key, resp.Header.Get(key))
		}
	}

	// body
	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return respHeader, nil, 0, nil
	}

	err = resp.Body.Close()
	if err != nil {
		return respHeader, nil, 0, nil
	}

	if len(respBodyBytes) > 0 {
		respBody = buffer.NewIoBufferBytes(respBodyBytes)
	}

	// trailer
	if len(resp.Trailer) > 0 {
		respTrailer = protocol.CommonHeader{}
		for key, _ := range resp.Trailer {
			respTrailer.Set(key, resp.Trailer.Get(key))
		}
	}

	return respHeader, respBody, len(respBodyBytes), respTrailer
}

// override
func (d *DefaultImportsHandler) GetHttpCallResponseHeaders() common.HeaderMap {
	if d.hc != nil && d.hc.respHeader != nil {
		return HeaderMapWrapper{HeaderMap: d.hc.respHeader}
	}

	return nil
}

// override
func (d *DefaultImportsHandler) GetHttpCallResponseBody() common.IoBuffer {
	if d.hc != nil && d.hc.respBody != nil {
		return IoBufferWrapper{IoBuffer: d.hc.respBody}
	}

	return nil
}

// override
func (d *DefaultImportsHandler) GetHttpCallResponseTrailer() common.HeaderMap {
	if d.hc != nil && d.hc.respTrailer != nil {
		return HeaderMapWrapper{HeaderMap: d.hc.respTrailer}
	}

	return nil
}
