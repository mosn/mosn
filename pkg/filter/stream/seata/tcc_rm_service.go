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
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/seata/seata-go/pkg/tm"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
)

type RequestContext struct {
	CommitPath   string                `tccParam:"MosnCommitPath"`
	RollbackPath string                `tccParam:"MosnRollbackPath"`
	VarHost      string                `tccParam:"MosnVarHost"`
	Query        string                `tccParam:"MosnQueryString"`
	Headers      protocol.CommonHeader `tccParam:"MosnQueryHeaders"`
	Trailers     protocol.CommonHeader `tccParam:"MosnQueryTrailers"`
	Body         string                `tccParam:"MosnQueryBody"`
}

type TccRMService struct {
	Name string
}

func (t *TccRMService) Prepare(ctx context.Context, params interface{}) (bool, error) {
	return true, nil
}

func (t *TccRMService) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	path, ok := businessActionContext.ActionContext["MosnCommitPath"]
	if !ok {
		log.DefaultLogger.Errorf("commit failed, businessActionContext: %v", businessActionContext)
		return false, errors.New("branch commit failed, miss param of MosnRollbackPath")
	}
	host, ok := businessActionContext.ActionContext["MosnVarHost"]
	if !ok {
		log.DefaultLogger.Errorf("commit failed, businessActionContext: %v", businessActionContext)
		return false, errors.New("branch commit failed, miss param of MosnVarHost")
	}
	query, ok := businessActionContext.ActionContext["MosnQueryString"]
	if !ok {
		log.DefaultLogger.Errorf("commit failed, businessActionContext: %v", businessActionContext)
		return false, errors.New("branch commit failed, miss param of MosnQueryString")
	}
	headers, ok := businessActionContext.ActionContext["MosnQueryHeaders"]
	if !ok {
		log.DefaultLogger.Errorf("commit failed, businessActionContext: %v", businessActionContext)
		return false, errors.New("branch commit failed, miss param of MosnQueryHeaders")
	}
	body, ok := businessActionContext.ActionContext["MosnQueryBody"]
	if !ok {
		log.DefaultLogger.Errorf("commit failed, businessActionContext: %v", businessActionContext)
		return false, errors.New("branch commit failed, miss param of MosnQueryBody")
	}

	resp, err := doHttp1Request(host.(string), path.(string), query.(string), headers.(map[string]interface{}), body)
	if err != nil {
		log.DefaultLogger.Errorf("commit failed, err: %v", err)
		return false, fmt.Errorf("branch commit failed, err: %v", err)
	}

	if resp.StatusCode() == http.StatusOK {
		return true, nil
	}
	return false, errors.New("branch commit failed")
}

func (t *TccRMService) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	path, ok := businessActionContext.ActionContext["MosnRollbackPath"]
	if !ok {
		log.DefaultLogger.Errorf("rollback failed, businessActionContext: %v", businessActionContext)
		return false, errors.New("branch rollback failed, miss param of MosnRollbackPath")
	}
	host, ok := businessActionContext.ActionContext["MosnVarHost"]
	if !ok {
		log.DefaultLogger.Errorf("rollback failed, businessActionContext: %v", businessActionContext)
		return false, errors.New("branch rollback failed, miss param of MosnVarHost")
	}
	query, ok := businessActionContext.ActionContext["MosnQueryString"]
	if !ok {
		log.DefaultLogger.Errorf("rollback failed, businessActionContext: %v", businessActionContext)
		return false, errors.New("branch rollback failed, miss param of MosnQueryString")
	}
	headers, ok := businessActionContext.ActionContext["MosnQueryHeaders"]
	if !ok {
		log.DefaultLogger.Errorf("rollback failed, businessActionContext: %v", businessActionContext)
		return false, errors.New("branch rollback failed, miss param of MosnQueryHeaders")
	}
	body, ok := businessActionContext.ActionContext["MosnQueryBody"]
	if !ok {
		log.DefaultLogger.Errorf("rollback failed, businessActionContext: %v", businessActionContext)
		return false, errors.New("branch rollback failed, miss param of MosnQueryBody")
	}

	resp, err := doHttp1Request(host.(string), path.(string), query.(string), headers.(map[string]interface{}), body)
	if err != nil {
		log.DefaultLogger.Errorf("rollback failed, err: %v", err)
		return false, fmt.Errorf("branch rollback failed, err: %v", err)
	}

	if resp.StatusCode() == http.StatusOK {
		return true, nil
	}
	return false, errors.New("branch rollback failed")
}

func (t *TccRMService) GetActionName() string {
	return t.Name
}

func doHttp1Request(host, path, queryString string, headers map[string]interface{}, body interface{}) (*resty.Response, error) {
	u := url.URL{
		Scheme: "http",
		Path:   path,
		Host:   host,
	}
	if len(queryString) > 0 {
		u.RawQuery = queryString
	}

	client := resty.New()
	request := client.R()
	for k, v := range headers {
		request.SetHeader(k, v.(string))
	}
	request.SetBody(body)
	return request.Post(u.String())
}
