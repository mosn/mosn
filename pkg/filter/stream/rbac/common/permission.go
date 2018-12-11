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

package common

import (
	"fmt"

	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/envoyproxy/go-control-plane/envoy/config/rbac/v2alpha"
)

type InheritPermission interface {
	Match(cb types.StreamReceiverFilterCallbacks, headers types.HeaderMap) bool
}

// PermissionAny
type PermissionAny struct {
	Any bool
}

func (permission *PermissionAny) Match(cb types.StreamReceiverFilterCallbacks, headers types.HeaderMap) bool {
	return permission.Any
}

// Receive the v2alpha.Permission input and convert it to mosn rbac permission
func NewInheritPermission(permission *v2alpha.Permission) (InheritPermission, error) {
	switch v := permission.Rule.(type) {
	case *v2alpha.Permission_Any:
		inheritPermission := new(PermissionAny)
		inheritPermission.Any = permission.Rule.(*v2alpha.Permission_Any).Any
		return inheritPermission, nil
	default:
		return nil, fmt.Errorf("not supported permission type found, detail: %v", v)
	}
}
