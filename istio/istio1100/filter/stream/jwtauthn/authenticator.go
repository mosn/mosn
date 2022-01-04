package jwtauthn

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

// Authenticator object to handle all JWT authentication flow.
type Authenticator interface {
	Verify(headers api.HeaderMap, tokens []JwtLocation) error
}

type authenticator struct {
	provider                string
	jwksCache               JwksCache
	jwksData                JwksData
	allowFail, allowMissing bool
	fetcher                 JwksFetcher

	tokens     []JwtLocation
	currToken  JwtLocation
	headers    api.HeaderMap
	jwtPayload string
}

func newAuthenticator(provider string, jwksCache JwksCache, fetcher JwksFetcher, allowFail, allowMissing bool) Authenticator {
	return &authenticator{
		provider:     provider,
		jwksCache:    jwksCache,
		fetcher:      fetcher,
		allowFail:    allowFail,
		allowMissing: allowMissing,
	}
}

func (a *authenticator) Verify(headers api.HeaderMap, tokens []JwtLocation) error {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Infof("JWT authentication starts, tokens size=%d", len(tokens))
	}
	if len(tokens) == 0 {
		return a.done(ErrJwtNotFound)
	}

	a.tokens = tokens
	a.headers = headers

	return a.startVerify()
}

func (a *authenticator) name() string {
	if a.provider != "" {
		optional := ""
		if a.allowMissing {
			optional = "-OPTIONAL"
		}
		return a.provider + optional
	}
	if a.allowFail {
		return "_IS_ALLOW_FALED_"
	}
	if a.allowMissing {
		return "_IS_ALLOW_MISSING_"
	}
	return "_UNKNOWN_"
}

func (a *authenticator) done(err error) error {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("%s: JWT token verification completed with: %v", a.name(), err)
	}

	// if on allow missing or failed this should verify all tokens, otherwise stop on ok.
	if len(a.tokens) == 0 {
		a.tokens = nil // clear tokens
		if a.allowFail {
			return nil
		}
		if a.allowMissing && err == ErrJwtNotFound {
			return nil
		}
		return err
	}

	if err == nil && !a.allowFail && !a.allowMissing {
		return nil
	}

	return a.startVerify()
}

func (a *authenticator) startVerify() error {
	a.currToken = a.tokens[0]
	a.tokens = a.tokens[1:]

	jwtToken, parts, err := new(jwt.Parser).ParseUnverified(a.currToken.Token(), &jwt.StandardClaims{})
	if err != nil {
		return a.done(ErrJwtBadFormat)
	}
	a.jwtPayload = parts[1]
	claims := jwtToken.Claims.(*jwt.StandardClaims)

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("Verifying JWT token of issuer: %s", claims.Issuer)
	}
	// Check if token extracted from the location contains the issuer specified by config.
	if !a.currToken.IsIssuerSpecified(claims.Issuer) {
		return a.done(ErrJwtUnknownIssuer)
	}

	// Check "exp" claim.
	if claims.ExpiresAt > 0 && claims.ExpiresAt < time.Now().Unix() {
		return a.done(ErrJwtExpired)
	}

	// Check "nbf" claim.
	if claims.NotBefore > time.Now().Unix() {
		return a.done(ErrJwtNotYetValid)
	}

	// Check the issuer is configured or not.
	if a.provider != "" {
		a.jwksData = a.jwksCache.FindByProvider(a.provider)
	} else {
		a.jwksData = a.jwksCache.FindByIssuer(claims.Issuer)
	}

	// Check if audience is allowed
	if !a.jwksData.AreAudiencesAllowed(claims.Audience) {
		return a.done(ErrJwtAudienceNotAllowed)
	}

	jwksObj := a.jwksData.GetJwksObj()
	if jwksObj != nil && !a.jwksData.IsExpired() {
		return a.done(a.verifyKey())
	}

	// Only one remote jwks will be fetched, verify will not continue util it is completed.
	if remoteJwks := a.jwksData.GetJwtProvider().GetRemoteJwks(); remoteJwks != nil {
		if a.fetcher == nil {
			a.fetcher = NewJwksFetcher()
		}

		jwks, err := a.fetcher.Fetch(remoteJwks.HttpUri)
		if err != nil {
			log.DefaultLogger.Errorf("fetch jwks: %v", err)
			return a.done(ErrJwksFetch)
		}
		a.jwksData.SetRemoteJwks(jwks)
		return a.done(a.verifyKey())
	}

	// No valid keys for this issuer. This may happen as a result of incorrect local JWKS configuration.
	return a.done(ErrJwksNoValidKeys)
}

func (a *authenticator) verifyKey() error {
	_, err := jwt.Parse(a.currToken.Token(), func(token *jwt.Token) (interface{}, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("expecting JWT header to have string kid")
		}

		keys := a.jwksData.GetJwksObj().LookupKeyID(kid)
		if len(keys) == 0 {
			return nil, fmt.Errorf("unable to find key %s", kid)
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
