package keyauth

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/any"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	pb "mosn.io/mosn/pkg/filter/stream/auth/keyauth/keyauthpb"
	"mosn.io/mosn/pkg/log"
)

// register create filter factory func
func init() {
	api.RegisterStream(v2.KeyAuth, CreateKeyAuthFilterFactory)
	api.RegisterXDSConfigHandler(v2.KeyAuth, func(s *any.Any) (map[string]interface{}, error) {
		m := new(pb.KeyAuth)
		if err := s.UnmarshalTo(m); err != nil { // transform `any` back to Struct
			log.DefaultLogger.Errorf("[KEYAUTH] convert fault inject config error: %v", err)
			return nil, err
		}
		newMap, err := StructToMap(m)
		if err != nil {
			log.DefaultLogger.Errorf("[KEYAUTH] convert config to map error: %v", err)
			return nil, err
		}
		return newMap, nil
	})
}

//StructToMap Converts a struct to a map while maintaining the json alias as keys
func StructToMap(obj interface{}) (newMap map[string]interface{}, err error) {
	data, err := json.Marshal(obj) // Convert to a json string

	if err != nil {
		return
	}

	err = json.Unmarshal(data, &newMap) // Convert to a map
	return
}

// FilterConfigFactory
type FilterConfigFactory struct {
	config *pb.KeyAuth
}

// CreateKeyAuthFilterFactory creates a new KeyAuthFilterFactory.
func CreateKeyAuthFilterFactory(cfg map[string]interface{}) (api.StreamFilterChainFactory, error) {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("create key_auth stream filter factory")
	}
	c, err := parseConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &FilterConfigFactory{
		config: c,
	}, nil
}

// CreateFilterChain creates a KeyAuthFilter
func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	pairs, err := NewMatcherVerifierPairs(f.config)
	if err != nil {
		log.DefaultLogger.Errorf("[key_auth filter] create new filter config: %v", err)
		return
	}

	filter := newAuthFilter(pairs)
	callbacks.AddStreamReceiverFilter(filter, api.AfterRoute)
}

// parseConfig parses cfg parse to *pb.KeyAuth
func parseConfig(cfg map[string]interface{}) (*pb.KeyAuth, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	filterConfig := &pb.KeyAuth{}
	if err := jsonpb.Unmarshal(bytes.NewReader(data), filterConfig); err != nil {
		return nil, err
	}

	return filterConfig, filterConfig.Validate()
}
