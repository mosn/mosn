package jwtauthn

import (
	"strings"

	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"github.com/valyala/fasthttp"
	"mosn.io/api"
)

const (
	//
	authorization = "Authorization"
	// The header value prefix for Authorization.
	bearerPrefix = "Bearer "
	// The default query parameter name to extract JWT token
	accessTokenParam = "access_token"
)

// JwtLocation stores following token information:
// 1. extracted token string
// 2. list of issuers specified the location.
type JwtLocation interface {
	// Get the token string
	Token() string
	// Check if an issuer has specified the location.
	IsIssuerSpecified(issuer string) bool
	// Remove the token from the headers
	RemoveJwt(headers api.HeaderMap)
}

// Extractor extracts JWT from locations specified in the config.
type Extractor interface {
	Extract(headers api.HeaderMap, requestArg string) []JwtLocation
}

// HeaderLocationSpec value type to store prefix and issuers that specified this header.
type HeaderLocationSpec struct {
	// The header name.
	headerName string
	// The value prefix. e.g. for "Bearer <token>", the value_prefix is "Bearer ".
	valuePrefix string
	// Issuers that specified this header.
	specifiedIssuers map[string]struct{}
}

func createHeaderLocationSpec(headerName, valuePrefix string) *HeaderLocationSpec {
	return &HeaderLocationSpec{
		headerName:       headerName,
		valuePrefix:      valuePrefix,
		specifiedIssuers: make(map[string]struct{}),
	}
}

func (h *HeaderLocationSpec) addIssuer(issuer string) {
	h.specifiedIssuers[issuer] = struct{}{}
}

// ParamLocationSpec value type to store issuers that specified this header.
type ParamLocationSpec struct {
	// Issuers that specified this param.
	specifiedIssuers map[string]struct{}
}

func createParamLocationSpec() *ParamLocationSpec {
	return &ParamLocationSpec{
		specifiedIssuers: make(map[string]struct{}),
	}
}

func (p *ParamLocationSpec) addIssuer(issuer string) {
	p.specifiedIssuers[issuer] = struct{}{}
}

// NewExtractor creates a new Extractor.
func NewExtractor(providers []*jwtauthnv3.JwtProvider) Extractor {
	extractor := &extractor{
		headerLocations: make(map[string]*HeaderLocationSpec),
		paramLocations:  make(map[string]*ParamLocationSpec),
	}
	extractor.addProviders(providers)
	return extractor
}

type extractor struct {
	headerLocations map[string]*HeaderLocationSpec
	paramLocations  map[string]*ParamLocationSpec
}

func (e *extractor) Extract(headers api.HeaderMap, requestArg string) []JwtLocation {
	var tokens []JwtLocation

	// Check header locations first.
	for _, hl := range e.headerLocations {
		token, ok := headers.Get(hl.headerName)
		if !ok {
			continue
		}
		if hl.valuePrefix != "" && !strings.HasPrefix(token, hl.valuePrefix) {
			// valuePrefix does not match
			continue
		}
		token = strings.Replace(token, hl.valuePrefix, "", 1)
		tokens = append(tokens, newJwtHeaderLocation(token, hl.headerName, hl.specifiedIssuers))
	}

	// If no query parameter locations specified, bail out
	if len(e.paramLocations) == 0 {
		return tokens
	}

	args := fasthttp.AcquireArgs()
	args.Parse(requestArg)
	// Check query parameter locations.
	for param, pl := range e.paramLocations {
		token := string(args.Peek(param))
		if token == "" {
			continue
		}
		tokens = append(tokens, newJwtParamLocation(token, pl.specifiedIssuers))
	}

	return tokens
}

func (e *extractor) addProviders(providers []*jwtauthnv3.JwtProvider) {
	for _, provider := range providers {
		e.addProvider(provider)
	}
}

func (e *extractor) addProvider(provider *jwtauthnv3.JwtProvider) {
	for _, header := range provider.GetFromHeaders() {
		e.addHeaderConfig(provider.GetIssuer(), header.GetName(), header.GetValuePrefix())
	}

	for _, param := range provider.GetFromParams() {
		e.addQueryParamConfig(provider.GetIssuer(), param)
	}

	// If not specified, use default locations.
	if len(provider.GetFromHeaders()) == 0 && len(provider.GetFromParams()) == 0 {
		e.addHeaderConfig(provider.GetIssuer(), authorization, bearerPrefix)
		e.addQueryParamConfig(provider.GetIssuer(), accessTokenParam)
	}

	// TODO(huangrh): forward payload headers
}

func (e *extractor) addHeaderConfig(issuer, headerName, valuePrefix string) {
	key := headerName + valuePrefix
	headerLocationSpec := e.headerLocations[key]
	if headerLocationSpec == nil {
		headerLocationSpec = createHeaderLocationSpec(headerName, valuePrefix)
		e.headerLocations[key] = headerLocationSpec
	}
	headerLocationSpec.addIssuer(issuer)
}

func (e *extractor) addQueryParamConfig(issuer, param string) {
	paramLocalSpec := e.paramLocations[param]
	if paramLocalSpec == nil {
		paramLocalSpec = createParamLocationSpec()
		e.paramLocations[param] = paramLocalSpec
	}
	paramLocalSpec.addIssuer(issuer)
}

type jwtLocationBase struct {
	token   string
	issuers map[string]struct{}
}

func (j *jwtLocationBase) IsIssuerSpecified(issuer string) bool {
	_, ok := j.issuers[issuer]
	return ok
}

func (j *jwtLocationBase) Token() string {
	return j.token
}

// jwtHeaderLocation is used for header extraction.
type jwtHeaderLocation struct {
	jwtLocationBase
	header string
}

// newJwtHeaderLocation creates a new jwtHeaderLocation.
func newJwtHeaderLocation(token, header string, issuers map[string]struct{}) JwtLocation {
	jhl := &jwtHeaderLocation{
		header: header,
	}
	jhl.token = token
	jhl.issuers = issuers
	return jhl
}

func (j *jwtHeaderLocation) RemoveJwt(headers api.HeaderMap) {
	headers.Del(j.header)
}

// JwtParamLocation is the JwtLocation for param extraction.
type JwtParamLocation struct {
	jwtLocationBase
}

// newJwtParamLocation creates a new jwtHeaderLocation.
func newJwtParamLocation(token string, issuers map[string]struct{}) JwtLocation {
	jpl := &JwtParamLocation{}
	jpl.token = token
	jpl.issuers = issuers
	return jpl
}

// RemoveJwt removes JWT from parameter
func (j *JwtParamLocation) RemoveJwt(headers api.HeaderMap) {
	// TODO(huangrh): remove JWT from parameter.
}
