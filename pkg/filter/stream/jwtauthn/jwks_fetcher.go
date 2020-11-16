package jwtauthn

import (
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/lestrrat/go-jwx/jwk"
)

// JwksFetcher is and interface can be used to retrieve remote JWKS.
type JwksFetcher interface {
	Fetch(uri *envoy_config_core_v3.HttpUri) (*jwk.Set, error)
}

type jwksFetcher struct {
}

// NewJwksFetcher creates a new JwksFetcher.
func NewJwksFetcher() JwksFetcher {
	return &jwksFetcher{}
}

func (j *jwksFetcher) Fetch(uri *envoy_config_core_v3.HttpUri) (*jwk.Set, error) {
	// Check if cluster is configured, fail the request if not.
	return nil, nil
}
