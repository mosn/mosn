package jwtauthn

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

// Authenticator object to handle all JWT authentication flow.
type Authenticator interface {
	Verify(headers api.HeaderMap, tokens []JwtLocation) error
}

type authenticator struct {
	provider   string
	jwksCache  JwksCache
	jwksData   JwksData
	fetcher    JwksFetcher
	tokens     []JwtLocation
	currToken  JwtLocation
	headers    api.HeaderMap
	jwt        *jwt.Token
	jwtPayload string
}

func newAuthenticator(provider string, jwksCache JwksCache, fetcher JwksFetcher, allowFail, allowMissing bool) Authenticator {
	return &authenticator{
		provider:  provider,
		jwksCache: jwksCache,
		fetcher:   fetcher,
	}
}

func newAuthenticatorDeprecated(config *jwtauthnv3.JwtAuthentication, fetcher JwksFetcher) Authenticator {
	return &authenticator{
		jwksCache: NewJwksCacheDeprecated(config),
		fetcher:   fetcher,
	}
}

func (a authenticator) Verify(headers api.HeaderMap, tokens []JwtLocation) error {
	log.DefaultLogger.Infof("JWT authentication starts, tokens size=%d", len(tokens))
	if len(tokens) == 0 {
		return ErrJwtNotFound
	}

	a.tokens = tokens
	a.headers = headers

	err := a.startVerify()
	if len(a.tokens) == 0 {
		return err
	}
	if err == nil {
		_ = a.Verify(headers, a.tokens)
		return nil
	}
	return a.Verify(headers, a.tokens)
}

func (a *authenticator) startVerify() error {
	a.currToken = a.tokens[0]
	a.tokens = a.tokens[1:]

	jwtToken, parts, err := new(jwt.Parser).ParseUnverified(a.currToken.Token(), &jwt.StandardClaims{})
	if err != nil {
		return ErrJwtBadFormat
	}
	a.jwtPayload = parts[1]
	claims := jwtToken.Claims.(*jwt.StandardClaims)

	log.DefaultLogger.Debugf("Verifying JWT token of issuer: %s", claims.Issuer)
	// Check if token extracted from the location contains the issuer specified by config.
	if !a.currToken.IsIssuerSpecified(claims.Issuer) {
		return ErrJwtUnknownIssuer
	}

	// Check "exp" claim.
	if claims.ExpiresAt > 0 && claims.ExpiresAt < time.Now().Unix() {
		return ErrJwtExpired
	}

	// Check "nbf" claim.
	if claims.NotBefore > time.Now().Unix() {
		return ErrJwtNotYetValid
	}

	// Check the issuer is configured or not.
	if a.provider != "" {
		a.jwksData = a.jwksCache.FindByProvider(a.provider)
	} else {
		a.jwksData = a.jwksCache.FindByIssuer(claims.Issuer)
	}

	// Check if audience is allowed
	if !a.jwksData.AreAudiencesAllowed(claims.Audience) {
		return ErrJwtAudienceNotAllowed
	}

	jwksObj := a.jwksData.GetJwksObj()
	if jwksObj != nil && !a.jwksData.IsExpired() {
		return a.verifyKey()
	}

	// Only one remote jwks will be fetched, verify will not continue util it is completed.
	if remoteJwks := a.jwksData.GetJwtProvider().GetRemoteJwks(); remoteJwks != nil {
		if a.fetcher == nil {
			a.fetcher = NewJwksFetcher()
		}

		jwks, err := a.fetcher.Fetch(remoteJwks.HttpUri)
		if err != nil {
			log.DefaultLogger.Errorf("fetch jwks: %v", err)
			return ErrJwksFetch
		}
		a.jwksData.SetRemoteJwks(jwks)
		return a.verifyKey()
	}

	// No valid keys for this issuer. This may happen as a result of incorrect local JWKS configuration.
	return ErrJwksNoValidKeys
}

func (a *authenticator) verifyKey() error {
	_, err := jwt.Parse(a.currToken.Token(), func(token *jwt.Token) (interface{}, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("expecting JWT header to have string kid")
		}

		keys := a.jwksData.GetJwksObj().LookupKeyID(kid)
		if len(keys) == 0 {
			return nil, fmt.Errorf("unable to find key %q", kid)
		}
		key := keys[0]

		return key.Materialize()
	})
	if err != nil {
		return ErrInvalidToken
	}

	provider := a.jwksData.GetJwtProvider()
	if !provider.Forward {
		// Remove JWT from headers.
		a.currToken.RemoveJwt(a.headers)
	}

	// Forward the payload
	if provider.ForwardPayloadHeader != "" {
		a.headers.Add(strings.ToLower(provider.ForwardPayloadHeader), a.jwtPayload)
	}

	if provider.PayloadInMetadata != "" {
		// TODO(huangrh)
	}

	return nil
}
