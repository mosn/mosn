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
	"reflect"

	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/envoyproxy/go-control-plane/envoy/config/rbac/v2alpha"
)

type InheritPermission interface {
	InheritPermission()
	// A policy matches if and only if at least one of InheritPermission.Match return true
	// AND at least one of InheritPrincipal.Match return true
	Match(cb types.StreamReceiverFilterCallbacks, headers types.HeaderMap) bool
}

func (*PermissionAny) InheritPermission() {}

// Permission_Any
type PermissionAny struct {
	Any bool
}

func NewPermissionAny(permission *v2alpha.Permission_Any) (*PermissionAny, error) {
	return &PermissionAny{
		Any: permission.Any,
	}, nil
}

func (permission *PermissionAny) Match(cb types.StreamReceiverFilterCallbacks, headers types.HeaderMap) bool {
	return permission.Any
}

// Receive the v2alpha.Permission input and convert it to mosn rbac permission
func NewInheritPermission(permission *v2alpha.Permission) (InheritPermission, error) {
	// Types that are valid to be assigned to Rule:
	//	*Permission_AndRules
	//	*Permission_OrRules
	//	*Permission_Any
	//	*Permission_Header
	//	*Permission_DestinationIp
	//	*Permission_DestinationPort
	//	*Permission_Metadata
	switch permission.Rule.(type) {
	case *v2alpha.Permission_Any:
		return NewPermissionAny(permission.Rule.(*v2alpha.Permission_Any))
	default:
		return nil, fmt.Errorf("[NewInheritPermission] not supported Permission.Rule type found, detail: %v",
			reflect.TypeOf(permission.Rule))
	}
}
