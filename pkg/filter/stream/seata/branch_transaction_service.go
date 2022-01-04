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
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gogo/protobuf/types"
	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"google.golang.org/grpc/metadata"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	mtypes "mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"
)

const (
	CommitRequestPath   = "tcc_commit_request_path"
	RollbackRequestPath = "tcc_rollback_request_path"
)

func (f *filter) branchCommunicate() {
	for {
		ctx := metadata.AppendToOutgoingContext(context.Background(), "addressing", f.conf.Addressing)
		stream, err := f.resourceClient.BranchCommunicate(ctx)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		done := make(chan bool)
		utils.GoWithRecover(func() {
			for {
				select {
				case _, ok := <-done:
					if !ok {
						return
					}
				case msg := <-f.branchMessages:
					err := stream.Send(msg)
					if err != nil {
						return
					}
				}
			}
		}, nil)

		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				close(done)
				break
			}
			if err != nil {
				close(done)
				break
			}
			switch msg.BranchMessageType {
			case apis.TypeBranchCommit:
				request := &apis.BranchCommitRequest{}
				data := msg.GetMessage().GetValue()
				err := request.Unmarshal(data)
				if err != nil {
					log.DefaultLogger.Errorf(err.Error(), nil)
					continue
				}
				response, err := branchCommit(context.Background(), request)
				if err == nil {
					content, err := types.MarshalAny(response)
					if err == nil {
						f.branchMessages <- &apis.BranchMessage{
							ID:                msg.ID,
							BranchMessageType: apis.TypeBranchCommitResult,
							Message:           content,
						}
					}
				}
			case apis.TypeBranchRollback:
				request := &apis.BranchRollbackRequest{}
				data := msg.GetMessage().GetValue()
				err := request.Unmarshal(data)
				if err != nil {
					log.DefaultLogger.Errorf(err.Error(), nil)
					continue
				}
				response, err := branchRollback(context.Background(), request)
				if err == nil {
					content, err := types.MarshalAny(response)
					if err == nil {
						f.branchMessages <- &apis.BranchMessage{
							ID:                msg.ID,
							BranchMessageType: apis.TypeBranchRollBackResult,
							Message:           content,
						}
					}
				}
			}
		}
		stream.CloseSend()
	}
}

// branchCommit commit branch transaction
func branchCommit(ctx context.Context, request *apis.BranchCommitRequest) (*apis.BranchCommitResponse, error) {
	requestContext := &RequestContext{
		ActionContext: make(map[string]string),
		Headers:       protocol.CommonHeader{},
		Body:          buffer.NewIoBuffer(0),
		Trailers:      protocol.CommonHeader{},
	}

	requestContext.Decode(request.ApplicationData)

	resp, err := doHttp1Request(requestContext, true)
	if err != nil {
		log.DefaultLogger.Errorf("commit failed, err: %v", err)
		return &apis.BranchCommitResponse{
			ResultCode: apis.ResultCodeFailed,
			Message:    err.Error(),
		}, nil
	}

	if resp.StatusCode() == http.StatusOK {
		return &apis.BranchCommitResponse{
			ResultCode:   apis.ResultCodeSuccess,
			XID:          request.XID,
			BranchID:     request.BranchID,
			BranchStatus: apis.PhaseTwoCommitted,
		}, nil
	}
	return &apis.BranchCommitResponse{
		ResultCode:   apis.ResultCodeSuccess,
		XID:          request.XID,
		BranchID:     request.BranchID,
		BranchStatus: apis.PhaseTwoCommitFailedRetryable,
	}, nil
}

// branchRollback rollback branch transaction
func branchRollback(ctx context.Context, request *apis.BranchRollbackRequest) (*apis.BranchRollbackResponse, error) {
	requestContext := &RequestContext{
		ActionContext: make(map[string]string),
		Headers:       protocol.CommonHeader{},
		Body:          buffer.NewIoBuffer(0),
		Trailers:      protocol.CommonHeader{},
	}

	requestContext.Decode(request.ApplicationData)

	resp, err := doHttp1Request(requestContext, false)
	if err != nil {
		log.DefaultLogger.Errorf("commit failed, err: %v", err)
		return &apis.BranchRollbackResponse{
			ResultCode: apis.ResultCodeFailed,
			Message:    err.Error(),
		}, nil
	}

	if resp.StatusCode() == http.StatusOK {
		return &apis.BranchRollbackResponse{
			ResultCode:   apis.ResultCodeSuccess,
			XID:          request.XID,
			BranchID:     request.BranchID,
			BranchStatus: apis.PhaseTwoRolledBack,
		}, nil
	}
	return &apis.BranchRollbackResponse{
		ResultCode:   apis.ResultCodeSuccess,
		XID:          request.XID,
		BranchID:     request.BranchID,
		BranchStatus: apis.PhaseTwoRollbackFailedRetryable,
	}, nil
}

func doHttp1Request(requestContext *RequestContext, commit bool) (*resty.Response, error) {
	var (
		host        string
		path        string
		queryString string
	)
	host = requestContext.ActionContext[mtypes.VarHost]
	if commit {
		path = requestContext.ActionContext[CommitRequestPath]
	} else {
		path = requestContext.ActionContext[RollbackRequestPath]
	}

	u := url.URL{
		Scheme: "http",
		Path:   path,
		Host:   host,
	}
	queryString, ok := requestContext.ActionContext[mtypes.VarQueryString]
	if ok {
		u.RawQuery = queryString
	}

	client := resty.New()
	request := client.R()
	requestContext.Headers.Range(func(key, value string) bool {
		request.SetHeader(key, value)
		return true
	})
	request.SetBody(requestContext.Body.Bytes())
	return request.Post(u.String())
}
