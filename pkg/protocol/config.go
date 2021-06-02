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

package protocol

import (
	"sync"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

// ProtocolConfigHandler translate general config into protocol configs
type ProtocolConfigHandler func(v interface{}) interface{}

var protocolConfigHandlers sync.Map

func RegisterProtocolConfigHandler(prot api.ProtocolName, h ProtocolConfigHandler) {
	_, loaded := protocolConfigHandlers.LoadOrStore(prot, h)
	if loaded {
		log.DefaultLogger.Errorf("protocol %s registered config handler failed, handler is already registered", prot)
	}
}

func HandleConfig(prot api.ProtocolName, v interface{}) interface{} {
	hv, ok := protocolConfigHandlers.Load(prot)
	if !ok {
		// log.DefaultLogger.Errorf("translate protocol %s config failed, no config handler registered", prot)
		return nil
	}
	handler, _ := hv.(ProtocolConfigHandler)
	return handler(v)
}
