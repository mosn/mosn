package jwtauthn

import (
	"fmt"

	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"mosn.io/api"
)

// Verifier supports verification of JWTs with configured requirements.
type Verifier interface {
	Verify(headers api.HeaderMap, requestArg string) error
}

// NewVerifier creates a new Verifier.
func NewVerifier(require *jwtauthnv3.JwtRequirement, providers map[string]*jwtauthnv3.JwtProvider, parentProviderNames []string, fetcher JwksFetcher) (Verifier, error) {
	parentProviders := make(map[string]*jwtauthnv3.JwtProvider)
	for _, name := range parentProviderNames {
		provider, exists := providers[name]
		if !exists {
			return nil, fmt.Errorf("required provider ['%s'] is not configured", name)
		}
		parentProviders[name] = provider
	}

	var providerName string
	switch {
	case require.GetProviderName() != "":
		providerName = require.GetProviderName()
	case require.GetRequiresAny() != nil:
		return newAnyVerifier(require.GetRequiresAny().GetRequirements(), providers, fetcher)
	case require.GetAllowMissing() != nil:
		return newAllowMissingVerifier(require, parentProviders, fetcher), nil
	}

	provider, exists := providers[providerName]
	if !exists {
		return nil, fmt.Errorf("required provider ['%s'] is not configured", providerName)
	}

	// TODO(huangrh): ProviderAndAudienceVerifier

	return newProviderVerifier(require, provider, fetcher), nil
}

type providerVerifier struct {
	providerName string
	extractor    Extractor
	jwksCache    JwksCache
	fetcher      JwksFetcher
}

func newProviderVerifier(require *jwtauthnv3.JwtRequirement, provider *jwtauthnv3.JwtProvider, fetcher JwksFetcher) *providerVerifier {
	jwksCache := NewJwksCache(map[string]*jwtauthnv3.JwtProvider{
		require.GetProviderName(): provider,
	})
	extractor := NewExtractor([]*jwtauthnv3.JwtProvider{provider})
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

// Base verifier for requires all or any.
type baseGroupVerifier struct {
	verifiers []Verifier
}

func (b *baseGroupVerifier) Verify(headers api.HeaderMap, requestArg string) error {
	var err error
	for _, verifier := range b.verifiers {
		err = verifier.Verify(headers, requestArg)
		if err == nil {
			return nil
		}
	}
	return err
}

// Requires any verifier.
type anyVerifier struct {
	baseGroupVerifier
}

func newAnyVerifier(requires []*jwtauthnv3.JwtRequirement, providers map[string]*jwtauthnv3.JwtProvider, fetcher JwksFetcher) (*anyVerifier, error) {
	var verifiers []Verifier
	var byPassTypeRequirement *jwtauthnv3.JwtRequirement
	var usedProviders []string
	for _, require := range requires {
		isRegularRequirement := true
		// TODO(huangrh): ProviderAndAudiences
		switch {
		case require.GetProviderName() != "":
			usedProviders = append(usedProviders, require.GetProviderName())
		case require.GetAllowMissing() != nil:
			// TODO(huangrh): AllowMissingOrFailed
			isRegularRequirement = false
			if byPassTypeRequirement == nil || byPassTypeRequirement.GetAllowMissing() != nil {
				// We need to keep only one by_pass_type_requirement. If both
				// kAllowMissing and kAllowMissingOrFailed are set, use
				// kAllowMissingOrFailed.
				byPassTypeRequirement = require
			}
		}
		if isRegularRequirement {
			verifier, err := NewVerifier(require, providers, nil, fetcher)
			if err != nil {
				return nil, err
			}
			verifiers = append(verifiers, verifier)
		}
	}

	if byPassTypeRequirement != nil {
		verifier, err := NewVerifier(byPassTypeRequirement, providers, usedProviders, fetcher)
		if err != nil {
			return nil, err
		}
		verifiers = append(verifiers, verifier)
	}

	anyVerifier := &anyVerifier{}
	anyVerifier.verifiers = verifiers
	return anyVerifier, nil
}

type allowMissingVerifier struct {
	providerName string
	extractor    Extractor
	jwksCache    JwksCache
	fetcher      JwksFetcher
}

func newAllowMissingVerifier(require *jwtauthnv3.JwtRequirement, providers map[string]*jwtauthnv3.JwtProvider, fetcher JwksFetcher) *allowMissingVerifier {
	// TODO(huangrh): use jwksCache from context
	jwksCache := NewJwksCache(providers)

	var provs []*jwtauthnv3.JwtProvider
	for _, provider := range providers {
		provs = append(provs, provider)
	}
	extractor := NewExtractor(provs)
	return &allowMissingVerifier{
		providerName: require.GetProviderName(),
		extractor:    extractor,
		jwksCache:    jwksCache,
		fetcher:      fetcher,
	}
}

func (a *allowMissingVerifier) Verify(headers api.HeaderMap, requestArg string) error {
	auth := newAuthenticator(a.providerName, a.jwksCache, a.fetcher, false, true)
	tokens := a.extractor.Extract(headers, requestArg)
	return auth.Verify(headers, tokens)
}
