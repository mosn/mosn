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

import (
	"github.com/alipay/sofa-mosn/pkg/istio/control"
	"github.com/gogo/protobuf/proto"
	"istio.io/api/mixer/v1/config/client"
)

// ServiceContext hold service config for HTTP
type ServiceContext struct {
	ClientContext *ClientContext

	// service config come from per filter config key "mixer"
	ServiceConfig *client.ServiceConfig
}

func NewServiceContext(clientContext *ClientContext, config *client.ServiceConfig) *ServiceContext {
	return &ServiceContext{
		ClientContext:clientContext,
		ServiceConfig:config,
	}
}

func (s *ServiceContext) AddStaticAttributes(requestContext *control.RequestContext) {
	s.ClientContext.AddLocalNodeAttributes(&requestContext.Attributes)

	if s.ClientContext.HasMixerConfig() {
		proto.Merge(&requestContext.Attributes, s.ClientContext.MixerAttributes())
	}
	if s.HasMixerConfig() {
		proto.Merge(&requestContext.Attributes, s.ServiceConfig.MixerAttributes)
	}
}

func (s *ServiceContext) HasMixerConfig() bool {
	return s.ServiceConfig != nil && s.ServiceConfig.MixerAttributes != nil && len(s.ServiceConfig.MixerAttributes.Attributes) > 0
}