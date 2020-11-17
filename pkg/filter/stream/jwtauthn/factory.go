package jwtauthn

import (
	"context"
	"encoding/json"
	"mosn.io/mosn/pkg/log"

	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

// register create filter factory func
func init() {
	api.RegisterStream(v2.JwtAuthn, CreateJwtAuthnFilterFactory)
}

// FilterConfigFactory -
type FilterConfigFactory struct {
	config *jwtauthnv3.JwtAuthentication
}

// CreateJwtAuthnFilterFactory creates a new JwtAuthnFilterFactory.
func CreateJwtAuthnFilterFactory(cfg map[string]interface{}) (api.StreamFilterChainFactory, error) {
	log.DefaultLogger.Debugf("create jwt authn stream filter factory")
	c, err := ParseJWTAuthnFilter(cfg)
	if err != nil {
		return nil, err
	}
	return &FilterConfigFactory{
		config: c,
	}, nil
}

// CreateFilterChain creates a JwtAuthnFilter
func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filterConfig := NewFilterConfig(f.config)
	filter := newJwtAuthnFilter(filterConfig)
	callbacks.AddStreamReceiverFilter(filter, api.AfterRoute)
}

// ParseJWTAuthnFilter parses cfg parse to *jwtauthnv3.JwtAuthentication
// TODO(huangrh: define a new struct or use jwtauthnv3.JwtAuthentication.
func ParseJWTAuthnFilter(cfg map[string]interface{}) (*jwtauthnv3.JwtAuthentication, error) {
	filterConfig := &jwtauthnv3.JwtAuthentication{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, filterConfig); err != nil {
		return nil, err
	}
	return filterConfig, nil
}
