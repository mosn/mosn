package router

import (
	"context"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/variable"
	"testing"
)

func TestVariableRouteRuleImpl(t *testing.T) {
	variable.RegisterVariable(variable.NewIndexedVariable("header", nil, nil, variable.BasicSetter, 0))
	virtualHostImpl := &VirtualHostImpl{virtualHostName: "test"}
	testCases := []struct {
		name     string
		value    string
		expected bool
	}{
		{"header", "test", true},
		{"header", "/test/test", false},
	}
	for i, tc := range testCases {
		route := &v2.Router{
			RouterConfig: v2.RouterConfig{
				Match: v2.RouterMatch{
					Variables: []v2.VariableMatcher{
						{
							"header",
							"test",
						},
					},
				},
				Route: v2.RouteAction{
					RouterActionConfig: v2.RouterActionConfig{
						ClusterName: "test",
					},
				},
			},
		}
		base, _ := NewRouteRuleImplBase(virtualHostImpl, route)
		m := make(map[string]string)
		for _, v := range route.Match.Variables {
			m[v.Name] = v.Value
		}
		ctx := variable.NewVariableContext(context.Background())
		rr := &VariableRouteRuleImpl{base, m}
		variable.SetVariableValue(ctx,tc.name,tc.value)
		result := rr.Match(ctx, protocol.CommonHeader(map[string]string{}), 1)
		if (result != nil) != tc.expected {
			t.Errorf("#%d want matched %v, but get matched %v\n", i, tc.expected, result)
		}
		if result != nil {
			if result.RouteRule().PathMatchCriterion().MatchType() != api.Variable {
				t.Errorf("#%d match type is not expected", i)
			}
		}

	}
}
