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

package grpc

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/variable"
)

const (
	grpcName      string = "gRPC"
	ServiceName   string = "serviceName"
	RequestResult string = "requestResult"
)

const (
	SUCCESS string = "success"
	FAIL    string = "fail"
)

var (
	builtinVariables = []variable.Variable{
		variable.NewStringVariable(grpcName+"_"+ServiceName, nil, nil, variable.DefaultStringSetter, 0),
	}
)

func init() {
	for idx := range builtinVariables {
		variable.Register(builtinVariables[idx])
	}

	variable.RegisterProtocolResource(api.ProtocolName(grpcName), api.PATH, ServiceName)
}
