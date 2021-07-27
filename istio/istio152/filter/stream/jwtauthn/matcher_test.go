package jwtauthn

import (
	"testing"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
)

func TestMatchPrefix(t *testing.T) {
	rule := &jwtauthnv3.RequirementRule{
		Match: &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_Prefix{
				Prefix: "/match",
			},
		},
	}

	matcher := NewMatcher(rule)
	assert.True(t, matcher.Matches(nil, "/match/this"))
	assert.False(t, matcher.Matches(nil, "/MATCH"))
	assert.True(t, matcher.Matches(nil, "/matching"))
	assert.False(t, matcher.Matches(nil, "/matc"))
	assert.False(t, matcher.Matches(nil, "/no"))
}

func TestMatchPrefixCaseInsensitive(t *testing.T) {
	rule := &jwtauthnv3.RequirementRule{
		Match: &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_Prefix{
				Prefix: "/match",
			},
			CaseSensitive: &wrappers.BoolValue{Value: false},
		},
	}

	matcher := NewMatcher(rule)
	assert.True(t, matcher.Matches(nil, "/matching"))
	assert.True(t, matcher.Matches(nil, "/MATCH"))
}

func TestMatchPath(t *testing.T) {
	rule := &jwtauthnv3.RequirementRule{
		Match: &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_Path{
				Path: "/match",
			},
			CaseSensitive: &wrappers.BoolValue{
				Value: false,
			},
		},
	}

	matcher := NewMatcher(rule)
	assert.True(t, matcher.Matches(nil, "/match"))
	assert.True(t, matcher.Matches(nil, "/MATCH"))
	assert.True(t, matcher.Matches(nil, "/match?ok=bye"))
	assert.False(t, matcher.Matches(nil, "/matc"))
	assert.False(t, matcher.Matches(nil, "/match/"))
	assert.False(t, matcher.Matches(nil, "/matching"))
}

func TestMatchHeader(t *testing.T) {
	rule := &jwtauthnv3.RequirementRule{
		Match: &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_Prefix{
				Prefix: "/",
			},
			Headers: []*routev3.HeaderMatcher{
				{
					Name:                 "Abc",
					HeaderMatchSpecifier: &routev3.HeaderMatcher_ExactMatch{},
				},
			},
		},
	}

	matcher := NewMatcher(rule)
	assert.True(t, matcher.Matches(newHeaders([2]string{"Abc", ""}), "/"))
	assert.True(t, matcher.Matches(newHeaders([2]string{"Abc", "some"}, [2]string{"b", ""}), "/"))
	assert.False(t, matcher.Matches(newHeaders([2]string{"Abcd", ""}), "/"))
	assert.False(t, matcher.Matches(nil, "/"))
	assert.False(t, matcher.Matches(newHeaders([2]string{"", ""}), "/"))
}
