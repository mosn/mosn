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

package http

import "github.com/alipay/sofa-mosn/pkg/istio/control"

type RequestHandler interface {
	Report(checkData *CheckData, reportData *ReportData)
}

type requestHandler struct {
	requestContext *control.RequestContext
}

func NewRequestHandler() RequestHandler {
	return &requestHandler{
		requestContext:control.NewRequestContext(),
	}
}

func (h *requestHandler) Report(checkData *CheckData, reportData *ReportData) {
	h.addForwardAttributes(checkData)
	h.addCheckAttributes(checkData)

	builder := newAttributesBuilder(h.requestContext)
	builder.ExtractReportAttributes(reportData)
}

func (h *requestHandler) addForwardAttributes(checkData *CheckData) {
	builder := newAttributesBuilder(h.requestContext)
	builder.ExtractForwardedAttributes(checkData)
}

func (h *requestHandler) addCheckAttributes(checkData *CheckData) {

}