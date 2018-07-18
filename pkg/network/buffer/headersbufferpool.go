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

package buffer

import (
	"sync"

	"github.com/alipay/sofa-mosn/pkg/types"
)

type headersBufferPoolV2 struct {
	sync.Pool
}

func (p *headersBufferPoolV2) Take(capacity int) (amap map[string]string) {
	v := p.Get()

	if v == nil {
		amap = make(map[string]string, capacity)
	} else {
		amap = v.(map[string]string)
	}

	return
}

func (p *headersBufferPoolV2) Give(amap map[string]string) {
	for k := range amap {
		delete(amap, k)
	}

	p.Put(amap)
}

func NewHeadersBufferPool(poolSize int) types.HeadersBufferPool {
	return &headersBufferPoolV2{}
}
