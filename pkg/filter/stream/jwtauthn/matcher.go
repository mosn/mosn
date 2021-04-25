package jwtauthn

import (
	"net/url"
	"strings"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

// Matcher supports matching a HTTP requests with JWT requirements.
type Matcher interface {
	Matches(headers api.HeaderMap, requestPath string) bool
}

// NewMatcher creates a new Matcher.
func NewMatcher(rule *jwtauthnv3.RequirementRule) Matcher {
	switch rule.GetMatch().PathSpecifier.(type) {
	case *routev3.RouteMatch_Prefix:
		return newPrefixMatcher(rule)

	default:
		// *routev3.RouteMatch_Path
		return newPathMatcher(rule)
	}
}

type baseMatcher struct {
	caseSensitive   bool
	headers         []*routev3.HeaderMatcher
	queryParameters []*routev3.QueryParameterMatcher
}

func newBaseMatcher(rule *jwtauthnv3.RequirementRule) *baseMatcher {
	// default case sensitive
	caseSensitive := true
	if rule.Match.CaseSensitive != nil {
		caseSensitive = rule.Match.CaseSensitive.Value
	}

	return &baseMatcher{
		caseSensitive:   caseSensitive,
		headers:         rule.GetMatch().GetHeaders(),
		queryParameters: rule.GetMatch().GetQueryParameters(),
	}
}

// Check match for HeaderMatcher and QueryParameterMatcher
func (p *baseMatcher) matchRoutes(headers api.HeaderMap, requestPath string) bool {
	matchs := matchHeaders(headers, p.headers)
	if len(p.queryParameters) > 0 {
		// TODO(huangrh): suppurt query parameters
	}
	return matchs
}

type prefixMatcher struct {
	prefix      string
	baseMatcher *baseMatcher
}

func newPrefixMatcher(rule *jwtauthnv3.RequirementRule) Matcher {
	baseMatcher := newBaseMatcher(rule)
	return &prefixMatcher{
		prefix:      rule.GetMatch().GetPrefix(),
		baseMatcher: baseMatcher,
	}
}

func (p *prefixMatcher) Matches(headers api.HeaderMap, requestPath string) bool {
	path, prefix := requestPath, p.prefix
	if !p.baseMatcher.caseSensitive {
		path, prefix = strings.ToLower(requestPath), strings.ToLower(p.prefix)
	}
	prefixMatch := strings.HasPrefix(path, prefix)

	if p.baseMatcher.matchRoutes(headers, requestPath) && prefixMatch {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("Prefix requirement '%s' matched.", p.prefix)
		}
		return true
	}
	return false
}

type pathMatcher struct {
	path          string
	caseSensitive bool
	baseMatcher   *baseMatcher
}

func newPathMatcher(rule *jwtauthnv3.RequirementRule) Matcher {
	baseMatcher := newBaseMatcher(rule)
	return &pathMatcher{
		path:          rule.GetMatch().GetPath(),
		caseSensitive: rule.GetMatch().GetCaseSensitive().Value,
		baseMatcher:   baseMatcher,
	}
}

func (p *pathMatcher) Matches(headers api.HeaderMap, requestPath string) bool {
	u, _ := url.Parse(requestPath)
	requestPath = u.Path

	pathMatch := p.path == requestPath
	if !p.caseSensitive {
		pathMatch = strings.EqualFold(strings.ToLower(p.path), strings.ToLower(requestPath))
	}
	if p.baseMatcher.matchRoutes(headers, requestPath) && pathMatch {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("Path requirement '%s' matched.", p.path)
		}
		return true
	}
	return false
}
