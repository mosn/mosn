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

package subprotocol

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var subProtocolFactories map[types.SubProtocol]CodecFactory

func init() {
	//subProtocolFactories = make(map[types.SubProtocol]CodecFactory)
}

// Register SubProtocol Plugin
func Register(prot types.SubProtocol, factory CodecFactory) {
	if subProtocolFactories == nil {
		subProtocolFactories = make(map[types.SubProtocol]CodecFactory)
	}
	subProtocolFactories[prot] = factory
}

// CreateSubProtocolCodec return SubProtocol Codec
func CreateSubProtocolCodec(context context.Context, prot types.SubProtocol) types.Multiplexing {

	if spc, ok := subProtocolFactories[prot]; ok {
		log.DefaultLogger.Tracef("create sub protocol codec %v success", prot)
		return spc.CreateSubProtocolCodec(context)
	}
	log.DefaultLogger.Errorf("unknown sub protocol = %v", prot)
	return nil
}
