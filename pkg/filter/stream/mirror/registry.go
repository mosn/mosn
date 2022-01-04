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

package mirror

import (
	"context"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

var (
	defaultAmplification = 1
	amplificationKey     = "amplification"
	broadcastKey         = "broadcast"
)

func init() {
	api.RegisterStream(v2.Mirror, NewMirrorConfig)
}

func NewMirrorConfig(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	c := &config{
		Amplification: defaultAmplification,
	}

	if ampValue, ok := conf[amplificationKey]; ok {
		if amp, ok := ampValue.(float64); ok && amp > 0 {
			c.Amplification = int(amp)
		}
	}
	if broadcast, ok := conf[broadcastKey]; ok {
		c.BroadCast = broadcast.(bool)
	}
	return c, nil
}

type config struct {
	Amplification int  `json:"amplification,omitempty"`
	BroadCast     bool `json:"broadcast,omitempty"`
}

func (c *config) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	m := &mirror{
		amplification: c.Amplification,
		broadcast:     c.BroadCast,
	}
	callbacks.AddStreamReceiverFilter(m, api.AfterRoute)
}
