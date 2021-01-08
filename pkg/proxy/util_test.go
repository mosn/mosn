package proxy

import (
	"context"
	"testing"
	"time"

	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
)

func TestParseProxyTimeout(t *testing.T) {
	var to Timeout
	headers := make(protocol.CommonHeader)
	ctx := variable.NewVariableContext(context.Background())

	parseProxyTimeout(ctx, &to, nil, headers)
	if to.TryTimeout != 0 || to.GlobalTimeout != types.GlobalTimeout {
		t.Errorf("parseProxyTimeout error")
	}

	variable.SetVariableValue(ctx, types.VarProxyTryTimeout, "10000")
	variable.SetVariableValue(ctx, types.VarProxyGlobalTimeout, "100000")

	parseProxyTimeout(ctx, &to, nil, headers)
	if to.TryTimeout != 10000*time.Millisecond || to.GlobalTimeout != 100000*time.Millisecond {
		t.Errorf("parseProxyTimeout error")
	}

	variable.SetVariableValue(ctx, types.VarProxyTryTimeout, "1000000")
	variable.SetVariableValue(ctx, types.VarProxyGlobalTimeout, "100000")

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
