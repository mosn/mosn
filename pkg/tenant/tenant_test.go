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

package tenant

import (
	"testing"
	"reflect"
	"sofastack.io/sofa-mosn/pkg/istio/mixerclient"
	"sofastack.io/sofa-mosn/pkg/featuregate"
)

func TestInit(t *testing.T) {
	featuregate.DefaultMutableFeatureGate.Set("")

	serviceNode := "||k1=v1~k2=v2~k3=v3"
	Init(serviceNode)
	tenantInfo := GetTenantInfo()

	excepted := map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"}
	if !reflect.DeepEqual(excepted, tenantInfo) {
		t.Errorf("excepted %+v, bug got %+v", excepted, tenantInfo)
	}

	g := mixerclient.GlobalList()
	l := len(g)
	if g[l-3] != "k1" || g[l-2] != "k2" || g[l-1] != "k3" {
		t.Errorf("expected k1, k2, k3, bug got %s, %s, %s", g[l-3], g[l-2], g[l-1])
	}
}