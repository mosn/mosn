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

package variable

import (
	"context"

	"mosn.io/api"
	"mosn.io/pkg/variable"

	"mosn.io/mosn/pkg/types"
)

// RegisterProtocolResource registers the resource as ProtocolResourceName
// forexample protocolVar[Http1+api.URI] = http_request_uri var
// Deprecated: use mosn.io/pkg/variable/protocolres.go:RegisterProtocolResource instead
func RegisterProtocolResource(protocol types.ProtocolName, resource api.ProtocolResourceName, varname string) error {
	return variable.RegisterProtocolResource(protocol, resource, varname)
}

// GetProtocolResource get URI,PATH,ARG var depends on ProtocolResourceName
// Deprecated: use mosn.io/pkg/variable/protocolres.go:GetProtocolResource instead
func GetProtocolResource(ctx context.Context, name api.ProtocolResourceName, data ...interface{}) (string, error) {
	return variable.GetProtocolResource(ctx, name, data...)
}
