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

package regulator

import (
	"context"

	"mosn.io/api"
)

func init() {
	// register default dimension
	RegisterNewDimension(defaultDimensionFunc)
}

type NewDimension func(info api.RequestInfo) InvocationDimension

var dimensionNewFunc NewDimension

func RegisterNewDimension(newDimension NewDimension) {
	dimensionNewFunc = newDimension
}

func GetNewDimensionFunc() NewDimension {
	return dimensionNewFunc
}

type defaultDimension struct {
	invocationKey string
	measureKey    string
}

func (d *defaultDimension) GetInvocationKey() string {
	return d.invocationKey
}

func (d *defaultDimension) GetMeasureKey() string {
	return d.measureKey
}

func defaultDimensionFunc(info api.RequestInfo) InvocationDimension {
	if info == nil || info.UpstreamHost() == nil || info.RouteEntry() == nil {
		return &defaultDimension{}
	} else {
		return &defaultDimension{
			invocationKey: info.UpstreamHost().AddressString(),
			measureKey:    info.RouteEntry().ClusterName(context.TODO()), //todo the future may not be uniform
		}
	}
}
