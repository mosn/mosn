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

package dubbo

import (
	"mosn.io/pkg/variable"
)

// [Protocol]: dubbo
const (
	dubboProtocolName      = "Dubbo_"
	VarDubboRequestService = dubboProtocolName + "service"
	VarDubboRequestMethod  = dubboProtocolName + "method"
)

var (
	buildinVariables = []variable.Variable{
		variable.NewStringVariable(VarDubboRequestService, nil, nil, variable.DefaultStringSetter, 0),
		variable.NewStringVariable(VarDubboRequestMethod, nil, nil, variable.DefaultStringSetter, 0),
	}
)

func Init(conf map[string]interface{}) {
	// variable must registry
	for idx := range buildinVariables {
		variable.Register(buildinVariables[idx])
	}

	// init subset key
	sskObj := conf[subsetKey]
	if sskObj == nil {
		return
	}

	ssk, ok := sskObj.(string)
	if !ok {
		return
	}
	podSubsetKey = ssk
	return
}
