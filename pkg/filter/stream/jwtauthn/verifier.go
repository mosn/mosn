package jwtauthn

import (
	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"mosn.io/api"
)

// Verifier supports verification of JWTs with configured requirements.
type Verifier interface {
	Verify(headers api.HeaderMap, requestArg string) error
}

// NewVerifier creates a new Verifier.
func NewVerifier(require *jwtauthnv3.JwtRequirement, providers map[string]*jwtauthnv3.JwtProvider, fetcher JwksFetcher) Verifier {
	return newProviderVerifier(require, providers, fetcher)
}

type providerVerifier struct {
	providerName string
	extractor    Extractor
	jwksCache    JwksCache
	fetcher      JwksFetcher
}

func newProviderVerifier(require *jwtauthnv3.JwtRequirement, providers map[string]*jwtauthnv3.JwtProvider, fetcher JwksFetcher) *providerVerifier {
	jwksCache := NewJwksCache(map[string]*jwtauthnv3.JwtProvider{
		require.GetProviderName(): providers[require.GetProviderName()],
	})
	var provs []*jwtauthnv3.JwtProvider
	provs = append(provs, providers[require.GetProviderName()])
	extractor := NewExtractor(provs)
	return &providerVerifier{
		providerName: require.GetProviderName(),
		extractor:    extractor,
		jwksCache:    jwksCache,
		fetcher:      fetcher,
	}
}

func (p *providerVerifier) Verify(headers api.HeaderMap, requestArg string) error {
	auth := newAuthenticator(p.providerName, p.jwksCache, p.fetcher, false, false)
	tokens := p.extractor.Extract(headers, requestArg)
	return auth.Verify(headers, tokens)
}
