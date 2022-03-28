package jwtauthn

import (
	"bytes"
	"context"
	"encoding/json"

	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"github.com/golang/protobuf/jsonpb"
	"mosn.io/api"
	"mosn.io/mosn/istio/istio1106/config/v2"
	"mosn.io/mosn/pkg/log"
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
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("create jwt authn stream filter factory")
	}
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
	filterConfig, err := NewFilterConfig(f.config)
	if err != nil {
		log.DefaultLogger.Errorf("[jwtauth filter] create new filter config: %v", err)
		return
	}
	filter := newJwtAuthnFilter(filterConfig)
	callbacks.AddStreamReceiverFilter(filter, api.AfterRoute)
}

// ParseJWTAuthnFilter parses cfg parse to *jwtauthnv3.JwtAuthentication
func ParseJWTAuthnFilter(cfg map[string]interface{}) (*jwtauthnv3.JwtAuthentication, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	filterConfig := &jwtauthnv3.JwtAuthentication{}
	if err := jsonpb.Unmarshal(bytes.NewReader(data), filterConfig); err != nil {
		return nil, err
	}

	return filterConfig, filterConfig.Validate()
}
