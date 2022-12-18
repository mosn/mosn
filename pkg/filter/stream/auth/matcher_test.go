package auth

import (
	"testing"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"mosn.io/api"
	pb "mosn.io/mosn/pkg/filter/stream/auth/matchpb"
	"mosn.io/mosn/pkg/protocol/http"
)

func TestMatchPrefix(t *testing.T) {

	config := &pb.RouteMatch{
		PathSpecifier: &pb.RouteMatch_Prefix{
			Prefix: "/match",
		},
	}

	matcher, _ := NewMatcher(config)
	assert.True(t, matcher.Matches(nil, "/match/this"))
	assert.False(t, matcher.Matches(nil, "/MATCH"))
	assert.True(t, matcher.Matches(nil, "/matching"))
	assert.False(t, matcher.Matches(nil, "/matc"))
	assert.False(t, matcher.Matches(nil, "/no"))
}

func TestMatchPrefixCaseInsensitive(t *testing.T) {

	config := &pb.RouteMatch{
		PathSpecifier: &pb.RouteMatch_Prefix{
			Prefix: "/match",
		},
		CaseSensitive: &wrappers.BoolValue{Value: false},
	}

	matcher, _ := NewMatcher(config)
	assert.True(t, matcher.Matches(nil, "/matching"))
	assert.True(t, matcher.Matches(nil, "/MATCH"))
}

func TestMatchPath(t *testing.T) {

	config := &pb.RouteMatch{
		PathSpecifier: &pb.RouteMatch_Path{
			Path: "/match",
		},
		CaseSensitive: &wrappers.BoolValue{Value: false},
	}

	matcher, _ := NewMatcher(config)
	assert.True(t, matcher.Matches(nil, "/match"))
	assert.True(t, matcher.Matches(nil, "/MATCH"))
	assert.True(t, matcher.Matches(nil, "/match?ok=bye"))
	assert.False(t, matcher.Matches(nil, "/matc"))
	assert.False(t, matcher.Matches(nil, "/match/"))
	assert.False(t, matcher.Matches(nil, "/matching"))
}

func TestMatchHeader(t *testing.T) {

	config := &pb.RouteMatch{
		PathSpecifier: &pb.RouteMatch_Prefix{
			Prefix: "/",
		},
		Headers: []*pb.HeaderMatcher{
			{
				Name:                 "name",
				HeaderMatchSpecifier: &pb.HeaderMatcher_ExactMatch{},
			},
		},
	}

	matcher, _ := NewMatcher(config)
	assert.True(t, matcher.Matches(newHeaders(map[string]string{"name": ""}), "/"))
	assert.True(t, matcher.Matches(newHeaders(map[string]string{"name": "some", "xx": ""}), "/"))
	assert.False(t, matcher.Matches(newHeaders(map[string]string{"name11": ""}), "/"))
	assert.False(t, matcher.Matches(nil, "/"))
	assert.False(t, matcher.Matches(newHeaders(map[string]string{"": ""}), "/"))
}

func newHeaders(headers map[string]string) api.HeaderMap {
	res := &http.RequestHeader{
		RequestHeader: &fasthttp.RequestHeader{},
	}
	for k, v := range headers {
		res.Set(k, v)
	}
	return res
}
