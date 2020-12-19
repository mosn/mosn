package proxy

import (
	"context"
	"testing"
	"time"

	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
)

func TestParseProxyTimeout(t *testing.T) {
	var to Timeout
	ctx := variable.NewVariableContext(context.Background())

	parseProxyTimeout(ctx, &to, nil)
	if to.TryTimeout != 0 || to.GlobalTimeout != types.GlobalTimeout {
		t.Errorf("parseProxyTimeout error")
	}

	variable.SetVariableValue(ctx, types.VarProxyTryTimeout, "10000")
	variable.SetVariableValue(ctx, types.VarProxyGlobalTimeout, "100000")

	parseProxyTimeout(ctx, &to, nil)
	if to.TryTimeout != 10000*time.Millisecond || to.GlobalTimeout != 100000*time.Millisecond {
		t.Errorf("parseProxyTimeout error")
	}

	variable.SetVariableValue(ctx, types.VarProxyTryTimeout, "1000000")
	variable.SetVariableValue(ctx, types.VarProxyGlobalTimeout, "100000")

	parseProxyTimeout(ctx, &to, nil)
	if to.TryTimeout != 0 || to.GlobalTimeout != 100000*time.Millisecond {
		t.Errorf("parseProxyTimeout error")
	}

	ctx = variable.NewVariableContext(context.Background())
	parseProxyTimeout(ctx, &to, &mockRoute{})
	if to.TryTimeout != time.Millisecond || to.GlobalTimeout != 10^6*time.Millisecond {
		t.Errorf("parseProxyTimeout error")
	}
}
