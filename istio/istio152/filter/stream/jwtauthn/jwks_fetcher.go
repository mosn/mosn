package jwtauthn

import (
	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/lestrrat/go-jwx/jwk"
)

// JwksFetcher is and interface can be used to retrieve remote JWKS.
type JwksFetcher interface {
	Fetch(uri *envoycorev3.HttpUri) (*jwk.Set, error)
}

type jwksFetcher struct {
}

// NewJwksFetcher creates a new JwksFetcher.
func NewJwksFetcher() JwksFetcher {
	return &jwksFetcher{}
}

func (j *jwksFetcher) Fetch(uri *envoycorev3.HttpUri) (*jwk.Set, error) {
	// Check if cluster is configured, fail the request if not.
	return jwk.Fetch(uri.GetUri())
}
