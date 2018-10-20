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

package control

import (
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/istio/mixerclient"
	"github.com/alipay/sofa-mosn/pkg/log"
	"istio.io/api/mixer/v1"
)

// ClientContextBase hold mixer client
type ClientContextBase struct {
	mixerClient mixerclient.MixerClient
	config *v2.Mixer
}

func NewClientContextBase(config *v2.Mixer) *ClientContextBase {
	log.DefaultLogger.Infof("report cluster: %s", config.Transport.ReportCluster)
	return &ClientContextBase{
		mixerClient:mixerclient.NewMixerClient(config.Transport.ReportCluster),
		config:config,
	}
}

func (c *ClientContextBase) SendReport(context *RequestContext) {
	c.mixerClient.Report(&context.Attributes)
}

func (c *ClientContextBase) HasMixerConfig() bool {
	return c.config != nil && c.config.MixerAttributes != nil && len(c.config.MixerAttributes.Attributes) > 0
}

func (c *ClientContextBase) MixerAttributes() *v1.Attributes {
	return c.config.MixerAttributes
}

func (c *ClientContextBase) AddLocalNodeAttributes(attributes *v1.Attributes) {

}