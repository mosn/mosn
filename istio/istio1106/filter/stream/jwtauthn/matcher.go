package jwtauthn

import (
	"net/url"
	"strings"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

// Matcher supports matching a HTTP requests with JWT requirements.
type Matcher interface {
	Matches(headers api.HeaderMap, requestPath string) bool
}

// NewMatcher creates a new Matcher.
func NewMatcher(match *routev3.RouteMatch) Matcher {
	switch match.PathSpecifier.(type) {
	case *routev3.RouteMatch_Prefix:
		return newPrefixMatcher(match)

	default:
		// *routev3.RouteMatch_Path
		return newPathMatcher(match)
	}
}

type baseMatcher struct {
	caseSensitive   bool
	headers         []*routev3.HeaderMatcher
	queryParameters []*routev3.QueryParameterMatcher
}

func newBaseMatcher(match *routev3.RouteMatch) *baseMatcher {
	// default case sensitive
	caseSensitive := true
	if match.CaseSensitive != nil {
		caseSensitive = match.CaseSensitive.Value
	}

	return &baseMatcher{
		caseSensitive:   caseSensitive,
		headers:         match.GetHeaders(),
		queryParameters: match.GetQueryParameters(),
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

func newPrefixMatcher(match *routev3.RouteMatch) Matcher {
	baseMatcher := newBaseMatcher(match)
	return &prefixMatcher{
		prefix:      match.GetPrefix(),
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
	path        string
	baseMatcher *baseMatcher
}

func newPathMatcher(match *routev3.RouteMatch) Matcher {
	baseMatcher := newBaseMatcher(match)
	return &pathMatcher{
		path:        match.GetPath(),
		baseMatcher: baseMatcher,
	}
}

func (p *pathMatcher) Matches(headers api.HeaderMap, requestPath string) bool {
	u, _ := url.Parse(requestPath)
	requestPath = u.Path

	pathMatch := p.path == requestPath
	if !p.baseMatcher.caseSensitive {
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
