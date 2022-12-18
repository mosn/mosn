package auth

import (
	"strings"

	"mosn.io/api"
	pb "mosn.io/mosn/pkg/filter/stream/auth/matchpb"
)

func matchHeaders(requestHeaders api.HeaderMap, configHeaders []*pb.HeaderMatcher) bool {
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

func matchHeader(requestHeaders api.HeaderMap, configHeader *pb.HeaderMatcher) bool {
	rHeaderValue, exists := requestHeaders.Get(configHeader.GetName())

	if !exists {
		return configHeader.InvertMatch && configHeader.GetPresentMatch()
	}

	match := false
	switch headerMatcher := configHeader.HeaderMatchSpecifier.(type) {
	case *pb.HeaderMatcher_ExactMatch:
		match = headerMatcher.ExactMatch == "" || headerMatcher.ExactMatch == rHeaderValue

	case *pb.HeaderMatcher_PresentMatch:
		match = true

	case *pb.HeaderMatcher_PrefixMatch:
		match = strings.HasPrefix(rHeaderValue, headerMatcher.PrefixMatch)

	case *pb.HeaderMatcher_SuffixMatch:
		match = strings.HasSuffix(rHeaderValue, headerMatcher.SuffixMatch)
	}

	return match != configHeader.InvertMatch
}
