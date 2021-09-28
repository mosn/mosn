package jwtauthn

import (
	"strings"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"mosn.io/api"
)

func matchHeaders(requestHeaders api.HeaderMap, configHeaders []*routev3.HeaderMatcher) bool {
	// No headers to match is considered a match.
	if len(configHeaders) == 0 {
		return true
	}

	if requestHeaders == nil {
		return false
	}

	for _, configHeader := range configHeaders {
		if !matchHeader(requestHeaders, configHeader) {
			return false
		}
	}

	return true
}

func matchHeader(requestHeaders api.HeaderMap, configHeader *routev3.HeaderMatcher) bool {
	rHeaderValue, exists := requestHeaders.Get(configHeader.GetName())

	if !exists {
		return configHeader.InvertMatch && configHeader.GetPresentMatch()
	}

	match := false
	switch headerMatcher := configHeader.HeaderMatchSpecifier.(type) {
	case *routev3.HeaderMatcher_ExactMatch:
		match = headerMatcher.ExactMatch == "" || headerMatcher.ExactMatch == rHeaderValue

	case *routev3.HeaderMatcher_SafeRegexMatch:
		// TODO(huangrh):

	case *routev3.HeaderMatcher_RangeMatch:
		// TODO(huangrh):

	case *routev3.HeaderMatcher_PresentMatch:
		match = true

	case *routev3.HeaderMatcher_PrefixMatch:
		match = strings.HasPrefix(rHeaderValue, headerMatcher.PrefixMatch)

	case *routev3.HeaderMatcher_SuffixMatch:
		match = strings.HasSuffix(rHeaderValue, headerMatcher.SuffixMatch)
	}

	return match != configHeader.InvertMatch
}
