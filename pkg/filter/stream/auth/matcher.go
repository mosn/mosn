package auth

import (
	"fmt"
	"net/url"
	"strings"

	"mosn.io/api"
	pb "mosn.io/mosn/pkg/filter/stream/auth/matchpb"
	"mosn.io/mosn/pkg/log"
)

// Matcher supports matching a HTTP requests with auth requirements.
type Matcher interface {
	Matches(headers api.HeaderMap, requestPath string) bool
}

// NewMatcher creates a new Matcher.
func NewMatcher(match *pb.RouteMatch) (Matcher, error) {
	switch match.PathSpecifier.(type) {
	case *pb.RouteMatch_Prefix:
		return newPrefixMatcher(match), nil

	case *pb.RouteMatch_Path:
		return newPathMatcher(match), nil
	}

	return nil, fmt.Errorf("Not supported Matcher type")
}

type baseMatcher struct {
	caseSensitive bool
	headers       []*pb.HeaderMatcher
}

func newBaseMatcher(match *pb.RouteMatch) *baseMatcher {
	// default case sensitive
	caseSensitive := true
	if match.CaseSensitive != nil {
		caseSensitive = match.CaseSensitive.Value
	}

	return &baseMatcher{
		caseSensitive: caseSensitive,
		headers:       match.GetHeaders(),
	}
}

// Check match for HeaderMatcher and QueryParameterMatcher
func (p *baseMatcher) matchRoutes(headers api.HeaderMap, requestPath string) bool {
	matchs := matchHeaders(headers, p.headers)
	return matchs
}

type prefixMatcher struct {
	prefix      string
	baseMatcher *baseMatcher
}

func newPrefixMatcher(match *pb.RouteMatch) Matcher {
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

func newPathMatcher(match *pb.RouteMatch) Matcher {
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
