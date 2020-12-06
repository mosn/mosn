package jwtauthn

import (
	"time"

	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"github.com/lestrrat/go-jwx/jwk"
	"mosn.io/mosn/pkg/log"
)

// PubkeyCacheExpirationSec is the default cache expiration time in 5 minutes.
const PubkeyCacheExpirationSec = 600

// JwksCache is an interface to access all configured Jwt rules and their cached Jwks objects.
// It only caches Jwks specified in the config.
type JwksCache interface {
	FindByIssuer(issuer string) JwksData
	FindByProvider(provider string) JwksData
}

// JwksData is an Interface to access a Jwks config rule and its cached Jwks object.
type JwksData interface {
	GetJwtProvider() *jwtauthnv3.JwtProvider
	GetJwksObj() *jwk.Set
	SetRemoteJwks(jwks *jwk.Set)
	IsExpired() bool
	AreAudiencesAllowed(aud string) bool
}

// NewJwksCache creates a new JwksCache.
func NewJwksCache(providers map[string]*jwtauthnv3.JwtProvider) JwksCache {
	issuer2JwksData := make(map[string]JwksData)
	provider2JwksData := make(map[string]JwksData)
	for name, provider := range providers {
		provider2JwksData[name] = NewJwksData(provider)
		if _, ok := issuer2JwksData[provider.GetIssuer()]; !ok {
			issuer2JwksData[provider.GetIssuer()] = provider2JwksData[name]
		}
	}

	return &jwksCache{
		issuer2JwksData:   issuer2JwksData,
		provider2JwksData: provider2JwksData,
	}
}

// NewJwksData creates a new JwksData.
func NewJwksData(provider *jwtauthnv3.JwtProvider) JwksData {
	jwksData := &jwksData{
		provider: provider,
	}

	if localJwks := provider.GetLocalJwks(); localJwks != nil {
		set, err := jwk.Parse([]byte(localJwks.GetInlineString()))
		if err != nil {
			log.DefaultLogger.Warnf("Invalid inline jwks for issuer: %s, jwks: %s", provider.GetIssuer(), localJwks)
		} else {
			// Never expire
			jwksData.setKey(set, time.Unix(1<<63-62135596801, 999999999))
		}
	}

	return jwksData
}

type jwksCache struct {
	issuer2JwksData   map[string]JwksData
	provider2JwksData map[string]JwksData
}

func (j *jwksCache) FindByIssuer(issuer string) JwksData {
	return j.issuer2JwksData[issuer]
}

func (j *jwksCache) FindByProvider(provider string) JwksData {
	return j.provider2JwksData[provider]
}

type jwksData struct {
	provider    *jwtauthnv3.JwtProvider
	jwks        *jwk.Set
	expiredTime time.Time
}

func (j *jwksData) GetJwtProvider() *jwtauthnv3.JwtProvider {
	return j.provider
}

func (j *jwksData) GetJwksObj() *jwk.Set {
	return j.jwks
}

func (j *jwksData) SetRemoteJwks(jwks *jwk.Set) {
	expiredTime := time.Now()
	if cacheDuration := j.provider.GetRemoteJwks().CacheDuration; cacheDuration != nil {
		expiredTime = expiredTime.Add(time.Duration(1) * time.Second)
	} else {
		expiredTime = expiredTime.Add(PubkeyCacheExpirationSec * time.Second)
	}

	j.setKey(jwks, expiredTime)
}

func (j *jwksData) IsExpired() bool {
	return j.expiredTime.Before(time.Now())
}

func (j *jwksData) AreAudiencesAllowed(aud string) bool {
	if len(j.provider.Audiences) == 0 {
		return true
	}
	for _, audience := range j.provider.Audiences {
		if aud == audience {
			return true
		}
	}
	return false
}

func (j *jwksData) setKey(jwks *jwk.Set, expire time.Time) {
	j.jwks = jwks
	j.expiredTime = expire
}
