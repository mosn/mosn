package router

import (
	"context"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/variable"
	"reflect"
	"regexp"
	"testing"
)

func TestVariableRouteRuleImpl(t *testing.T) {
	variable.RegisterVariable(variable.NewIndexedVariable("header", nil, nil, variable.BasicSetter, 0))
	variable.RegisterVariable(variable.NewIndexedVariable("method", nil, nil, variable.BasicSetter, 0))
	variable.RegisterVariable(variable.NewIndexedVariable("uri", nil, nil, variable.BasicSetter, 0))

	virtualHostImpl := &VirtualHostImpl{virtualHostName: "test"}
	testCases := []struct {
		name     string
		value    string
		expected bool
	}{
		{"header", "test", true},
		{"header", "/test/test", false},
		{"method", "test", true},
		{"uri", "/1234", true},
		{"uri", "/abc", false},
	}
	route := &v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{
				Variables: []v2.VariableMatcher{
					{
						Name:  "header",
						Value: "test",
						Model: "or",
					},
					{
						Name:  "method",
						Value: "test",
						Model: "or",
					},
					{
						Name:  "uri",
						Regex: "/[0-9]+",
						Model: "and",
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
	for i, tc := range testCases {
		base, _ := NewRouteRuleImplBase(virtualHostImpl, route)
		variables := make([]*VariableMatchItem, len(route.Match.Variables))
		for i := range route.Match.Variables {
			variables[i] = ParseToVariableMatchItem(route.Match.Variables[i])
		}

		ctx := variable.NewVariableContext(context.Background())
		rr := &VariableRouteRuleImpl{base, variables}
		variable.SetVariableValue(ctx, tc.name, tc.value)
		result := rr.Match(ctx, protocol.CommonHeader(map[string]string{}))
		assert.EqualValuesf(t, result != nil, tc.expected, "#%d want matched %v, but get matched %v\n", i, tc.expected, result)
		if result != nil {
			assert.EqualValuesf(t, api.Variable, result.RouteRule().PathMatchCriterion().MatchType(), "#%d match type is not expected", i)
		}
	}
}

func TestParseToVariableMatchItem(t *testing.T) {
	s := func(s string) *string { return &s }
	reg, _ := regexp.Compile("/[0-9]+")
	type args struct {
		matcher v2.VariableMatcher
	}
	tests := []struct {
		name string
		args args
		want *VariableMatchItem
	}{
		{
			name: "p1",
			args: args{
				matcher: v2.VariableMatcher{
					Name:  "p1",
					Value: "test1",
				},
			},
			want: &VariableMatchItem{
				name:         "p1",
				value:        s("test1"),
				regexPattern: nil,
				model:        AND,
			},
		},
		{
			name: "p2",
			args: args{
				matcher: v2.VariableMatcher{
					Name:  "p2",
					Value: "test2",
					Model: "Or",
				},
			},
			want: &VariableMatchItem{
				name:         "p2",
				value:        s("test2"),
				regexPattern: nil,
				model:        OR,
			},
		},
		{
			name: "p3",
			args: args{
				matcher: v2.VariableMatcher{
					Name:  "p3",
					Model: "and",
					Regex: "/[0-9]+",
				},
			},
			want: &VariableMatchItem{
				name:         "p3",
				regexPattern: reg,
				model:        AND,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseToVariableMatchItem(tt.args.matcher); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseToVariableMatchItem() = %v, want %v", got, tt.want)
			}
		})
	}
}
