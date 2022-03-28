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

import "mosn.io/mosn/istio/istio152/istio/control"

// RequestHandler handle a HTTP request
type RequestHandler interface {
	Report(checkData CheckData, reportData ReportData)
}

type requestHandler struct {
	requestContext         *control.RequestContext
	serviceContext         *ServiceContext
	forwardAttributesAdded bool
	checkAttributesAdded   bool
}

// NewRequestHandler return RequestHandler
func NewRequestHandler(serviceContext *ServiceContext) RequestHandler {
	return &requestHandler{
		requestContext: control.NewRequestContext(),
		serviceContext: serviceContext,
	}
}

// Report Make a Report call. It will:
// * check service config to see if Report is required
// * extract check attributes if not done yet.
// * extract more report attributes
// * make a Report call.
func (h *requestHandler) Report(checkData CheckData, reportData ReportData) {
	if h.serviceContext != nil && h.serviceContext.serviceConfig != nil && h.serviceContext.serviceConfig.DisableReportCalls {
		return
	}
	h.addForwardAttributes(checkData)
	h.addCheckAttributes(checkData)

	builder := newAttributesBuilder(h.requestContext)
	builder.ExtractReportAttributes(reportData)

	h.serviceContext.GetClientContext().SendReport(h.requestContext)
}

func (h *requestHandler) addForwardAttributes(checkData CheckData) {
	if h.forwardAttributesAdded {
		return
	}
	h.forwardAttributesAdded = true
	builder := newAttributesBuilder(h.requestContext)
	builder.ExtractForwardedAttributes(checkData)
}

func (h *requestHandler) addCheckAttributes(checkData CheckData) {
	if h.checkAttributesAdded {
		return
	}
	h.checkAttributesAdded = true
	h.serviceContext.AddStaticAttributes(h.requestContext)

	builder := newAttributesBuilder(h.requestContext)
	builder.ExtractCheckAttributes(checkData)
	// TODO: AddApiAttributes
}
