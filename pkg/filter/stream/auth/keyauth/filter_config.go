package keyauth

import (
	"mosn.io/mosn/pkg/filter/stream/auth"
	pb "mosn.io/mosn/pkg/filter/stream/auth/keyauth/keyauthpb"
)

// NewFilterConfig creates a new filter config.
func NewMatcherVerifierPairs(config *pb.KeyAuth) ([]*MatcherVerifierPair, error) {

	authenticators := make(map[string]auth.Authenticator, len(config.Authenticators))
	for name, authConfig := range config.Authenticators {
		authenticators[name] = NewKeyAuth(authConfig)
	}

	MatcherVerifiers := make([]*MatcherVerifierPair, len(config.Rules))
	for i, rule := range config.Rules {
		verifier, err := auth.NewVerifier(rule.GetRequires(), authenticators)
		if err != nil {
			return nil, err
		}

		matcher, err := auth.NewMatcher(rule.GetMatch())
		if err != nil {
			return nil, err
		}

		MatcherVerifiers[i] = &MatcherVerifierPair{
			Matcher:  matcher,
			Verifier: verifier,
		}
	}

	return MatcherVerifiers, nil
}

// MatcherVerifierPair is a pair of matcher and Verifier.
type MatcherVerifierPair struct {
	Matcher  auth.Matcher
	Verifier auth.Verifier
}
