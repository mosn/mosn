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

package proxy

import (
	"context"
	"testing"
	"time"

	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

func TestParseProxyTimeout(t *testing.T) {
	var to Timeout
	headers := make(protocol.CommonHeader)
	ctx := variable.NewVariableContext(context.Background())

	parseProxyTimeout(ctx, &to, nil, headers)
	if to.TryTimeout != 0 || to.GlobalTimeout != types.GlobalTimeout {
		t.Errorf("parseProxyTimeout error")
	}

	variable.SetString(ctx, types.VarProxyTryTimeout, "10000")
	variable.SetString(ctx, types.VarProxyGlobalTimeout, "100000")

	parseProxyTimeout(ctx, &to, nil, headers)
	if to.TryTimeout != 10000*time.Millisecond || to.GlobalTimeout != 100000*time.Millisecond {
		t.Errorf("parseProxyTimeout error")
	}

	variable.SetString(ctx, types.VarProxyTryTimeout, "1000000")
	variable.SetString(ctx, types.VarProxyGlobalTimeout, "100000")

	parseProxyTimeout(ctx, &to, nil, headers)
	if to.TryTimeout != 0 || to.GlobalTimeout != 100000*time.Millisecond {
		t.Errorf("parseProxyTimeout error")
	}

	ctx = variable.NewVariableContext(context.Background())
	parseProxyTimeout(ctx, &to, &mockRoute{}, headers)
	if to.TryTimeout != time.Millisecond || to.GlobalTimeout != 10^6*time.Millisecond {
		t.Errorf("parseProxyTimeout error")
	}

	headers.Set(types.HeaderGlobalTimeout, "1000")
	headers.Set(types.HeaderTryTimeout, "100")
	parseProxyTimeout(ctx, &to, nil, headers)
	if to.TryTimeout != 100*time.Millisecond || to.GlobalTimeout != 1000*time.Millisecond {
		t.Errorf("parseProxyTimeout error")
	}
}
