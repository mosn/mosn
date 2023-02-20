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

package seata

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/valyala/fasthttp"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	mosnhttp "mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"
	"mosn.io/pkg/variable"
)

const (
	SEATA        = "seata"
	XID          = "x_seata_xid"
	ResponseFlag = 0x600
)

var (
	onceSeataInit    = &sync.Once{}
	transactionInfos = make(map[string]*TransactionInfo)
	tccResources     = make(map[string]*TCCResource)
	tccProxys        = make(map[string]*tcc.TCCServiceProxy)
)

type filter struct {
	conf           *Seata
	receiveHandler api.StreamReceiverFilterHandler
	sendHandler    api.StreamSenderFilterHandler
	globalRollback chan bool
	xid            chan string
}

func NewFilter(conf *Seata) (*filter, error) {
	var err error
	onceSeataInit.Do(func() {
		client.InitPath(conf.ConfPath)
		for _, ti := range conf.TransactionInfos {
			transactionInfos[ti.RequestPath] = ti
		}

		for _, r := range conf.TCCResources {
			proxy, e := tcc.NewTCCServiceProxy(&TccRMService{Name: conf.Name})
			if e != nil {
				err = e
			}
			tccResources[r.PrepareRequestPath] = r
			tccProxys[r.PrepareRequestPath] = proxy
		}
	})
	if err != nil {
		return nil, err
	}

	f := &filter{
		conf:           conf,
		globalRollback: make(chan bool),
		xid:            make(chan string),
	}

	return f, nil
}

func (f *filter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	if f.receiveHandler.RequestInfo().Protocol() == protocol.HTTP1 {
		path, err := variable.GetString(ctx, types.VarPath)
		if err != nil {
			log.Proxy.Errorf(ctx, "failed to get request path, err: %v", err)
			f.abort(ctx, http.StatusNotAcceptable, headers, `{"error":"failed to get request path"}`)
			return api.StreamFilterStop
		}
		method, err := variable.GetString(ctx, types.VarMethod)
		if err != nil {
			log.Proxy.Errorf(ctx, "failed to get method, err: %v", err)
			f.abort(ctx, http.StatusMethodNotAllowed, headers, `{"error":"failed to get http method"}`)
			return api.StreamFilterStop
		}

		if method != fasthttp.MethodPost {
			return api.StreamFilterContinue
		}

		transactionInfo, found := transactionInfos[strings.ToLower(path)]
		if found {
			return f.handleHttp1GlobalBegin(ctx, headers, buf, trailers, transactionInfo)
		}

		tccResource, exists := tccResources[strings.ToLower(path)]
		if exists {
			return f.handleHttp1BranchRegister(ctx, headers, buf, trailers, tccResource)
		}
	}
	return api.StreamFilterContinue
}

func (f *filter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	if f.receiveHandler.RequestInfo().Protocol() == protocol.HTTP1 {
		path, err := variable.GetString(ctx, types.VarPath)
		if err != nil {
			return api.StreamFilterContinue
		}
		method, err := variable.GetString(ctx, types.VarMethod)
		if err != nil {
			return api.StreamFilterContinue
		}

		if method != fasthttp.MethodPost {
			return api.StreamFilterContinue
		}

		_, found := transactionInfos[strings.ToLower(path)]
		if found {
			header, ok := headers.(mosnhttp.ResponseHeader)
			if ok {
				if header.StatusCode() == http.StatusOK {
					f.globalRollback <- false
					return api.StreamFilterContinue
				}
			}
			f.globalRollback <- true
		}
	}

	return api.StreamFilterContinue
}

// SetReceiveFilterHandler sets decoder filter callbacks
func (f *filter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.receiveHandler = handler
}

func (f *filter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.sendHandler = handler
}

func (f *filter) OnDestroy() {
}

func (f *filter) handleHttp1GlobalBegin(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer,
	trailers api.HeaderMap, transactionInfo *TransactionInfo) api.StreamFilterStatus {

	utils.GoWithRecover(func() {
		tm.WithGlobalTx(
			context.Background(),
			&tm.GtxConfig{
				Name: f.conf.Name,
			},
			func(ctx context.Context) (re error) {
				f.xid <- tm.GetXID(ctx)
				r := <-f.globalRollback
				if r {
					re = errors.Errorf("Global Roolback")
				}
				return
			})
	}, nil)

	xid := <-f.xid
	headers.Add(XID, xid)

	return api.StreamFilterContinue
}

func (f *filter) handleHttp1BranchRegister(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer,
	trailers api.HeaderMap, tccResource *TCCResource) api.StreamFilterStatus {
	xid, ok := headers.Get(XID)
	if !ok {
		log.Proxy.Errorf(ctx, "failed to get xid from request header")
		f.abort(ctx, http.StatusInternalServerError, headers,
			`{"error":"failed to get xid from request header"}`)
		return api.StreamFilterStop
	}

	requestContext := RequestContext{
		Trailers: make(map[string]string),
		Headers:  make(map[string]string),
		Body:     buf.Clone().String(),
	}
	host, _ := variable.GetString(ctx, types.VarHost)
	requestContext.VarHost = host
	requestContext.CommitPath = tccResource.CommitRequestPath
	requestContext.RollbackPath = tccResource.RollbackRequestPath
	queryString, _ := variable.GetString(ctx, types.VarQueryString)
	if queryString != "" {
		requestContext.Query = queryString
	}
	if headers != nil {
		headers.Range(func(key, value string) bool {
			requestContext.Headers.Set(key, value)
			return true
		})
	}
	if trailers != nil {
		trailers.Range(func(key, value string) bool {
			requestContext.Trailers.Set(key, value)
			return true
		})
	}

	newCtx := tm.InitSeataContext(ctx)
	tm.SetXID(newCtx, xid)
	tccProxy := tccProxys[tccResource.PrepareRequestPath]
	_, err := tccProxy.Prepare(newCtx, requestContext)
	if err != nil {
		log.Proxy.Errorf(ctx, "branch transaction register failed, xid: %s, err: %v", xid, err)
		f.abort(ctx, http.StatusInternalServerError, headers,
			fmt.Sprintf(`{"error":"branch transaction register failed, %v"}`, err))
		return api.StreamFilterStop
	}

	return api.StreamFilterContinue
}

func (f *filter) abort(ctx context.Context, code int, headers api.HeaderMap, body string) {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(ctx, "[stream filter] [seata] abort")
	}
	f.receiveHandler.RequestInfo().SetResponseFlag(ResponseFlag)
	f.receiveHandler.SendHijackReplyWithBody(code, headers, body)
}
