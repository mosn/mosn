package xds

import (
	"errors"
	"fmt"
	"strings"
	"time"

	envoy_config_bootstrap_v2 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	envoy_config_bootstrap_v3 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/duration"
	jsoniter "github.com/json-iterator/go"
	mv2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	v2conv "mosn.io/mosn/pkg/xds/v2/conv"
	v3conv "mosn.io/mosn/pkg/xds/v3/conv"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func duration2String(duration *duration.Duration) string {
	d := time.Duration(duration.Seconds)*time.Second + time.Duration(duration.Nanos)*time.Nanosecond
	x := fmt.Sprintf("%.9f", d.Seconds())
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, "000")
	return x + "s"
}

// Client XdsClient
type Client interface {
	Start() error
	Stop()
}

// NewClient build xds Client
func NewClient(config *mv2.MOSNConfig) (Client, error) {
	dynamicResources, err := unmarshalDynamicResource(config)
	if err != nil {
		log.DefaultLogger.Warnf("fail to unmarshal xds dynamic resources, skip xds: %v", err)
		return nil, errors.New("fail to unmarshal xds dynamic resources")
	}
	if dynamicResources == nil {
		return nil, errors.New("dynamicResource is empty")
	}

	switch types.XdsVersion {
	case types.XdsVersionV3:
		d, ok := dynamicResources.(*envoy_config_bootstrap_v3.Bootstrap_DynamicResources)
		if ok {
			staticResources, err := unmarshalStaticResourcesV3(config)
			if err != nil {
				log.DefaultLogger.Warnf("fail to unmarshal xds v3 static resources, skip xds: %v", err)
				return nil, errors.New("fail to unmarshal xds v3 static resources")
			}
			return &clientv3{config: config, dynamicResources: d, staticResources: staticResources}, nil
		}

	case types.XdsVersionV2:
		d, ok := dynamicResources.(*envoy_config_bootstrap_v2.Bootstrap_DynamicResources)
		if ok {
			staticResources, err := unmarshalStaticResourcesV2(config)
			if err != nil {
				log.DefaultLogger.Warnf("fail to unmarshal xds v2 static resources, skip xds: %v", err)
				return nil, errors.New("fail to unmarshal xds v2 static resources")
			}
			return &clientv2{config: config, dynamicResources: d, staticResources: staticResources}, nil
		}
	default:
	}

	// not do this
	return nil, fmt.Errorf("xds %s deal failed, check dynamicResource %s or staticResources %s", types.XdsVersion, config.RawDynamicResources, config.RawStaticResources)
}

// InitStats init stats
func InitStats() {
	switch types.XdsVersion {
	case types.XdsVersionV3:
		v3conv.InitStats()
	default:
		v2conv.InitStats()
	}
}

// GetStats return xdsStats
func GetStats() types.XdsStats {
	switch types.XdsVersion {
	case types.XdsVersionV3:
		return v3conv.Stats
	default:
		return v2conv.Stats
	}
}

func unmarshalDynamicResource(config *mv2.MOSNConfig) (dynamicResources interface{}, err error) {
	if len(config.RawDynamicResources) < 1 {
		return nil, nil
	}

	resources := map[string]jsoniter.RawMessage{}
	err = json.Unmarshal(config.RawDynamicResources, &resources)
	if err != nil {
		log.DefaultLogger.Errorf("fail to unmarshal dynamic_resources: %v", err)
		return nil, err
	}

	adsConfigRaw, ok := resources["ads_config"]
	if !ok {
		log.DefaultLogger.Errorf("ads_config not found")
		return nil, errors.New("lack of ads_config")
	}

	var b []byte
	adsConfig := map[string]jsoniter.RawMessage{}
	err = json.Unmarshal([]byte(adsConfigRaw), &adsConfig)
	if err != nil {
		log.DefaultLogger.Errorf("fail to unmarshal ads_config: %v", err)
		return nil, err
	}
	if refreshDelayRaw, ok := adsConfig["refresh_delay"]; ok {
		refreshDelay := duration.Duration{}
		err = json.Unmarshal([]byte(refreshDelayRaw), &refreshDelay)
		if err != nil {
			log.DefaultLogger.Errorf("fail to unmarshal refresh_delay: %v", err)
			return nil, err
		}

		d := duration2String(&refreshDelay)
		b, err = json.Marshal(&d)
		adsConfig["refresh_delay"] = jsoniter.RawMessage(b)
	}
	if transportAPIVersionRaw, ok := adsConfig["transport_api_version"]; ok {
		var transportAPIVersion string
		err = json.Unmarshal([]byte(transportAPIVersionRaw), &transportAPIVersion)
		if err != nil {
			log.DefaultLogger.Errorf("fail to unmarshal transport_api_version: %v", err)
			return nil, err
		}
		if strings.EqualFold(transportAPIVersion, types.XdsVersionV3) {
			types.XdsVersion = types.XdsVersionV3
		}
	}
	b, err = json.Marshal(&adsConfig)
	if err != nil {
		log.DefaultLogger.Errorf("fail to marshal refresh_delay: %v", err)
		return nil, err
	}
	resources["ads_config"] = jsoniter.RawMessage(b)
	b, err = json.Marshal(&resources)
	if err != nil {
		log.DefaultLogger.Errorf("fail to marshal ads_config: %v", err)
		return nil, err
	}

	if types.XdsVersion == types.XdsVersionV3 {
		dynamicResourcesV3 := &envoy_config_bootstrap_v3.Bootstrap_DynamicResources{}
		err = jsonpb.UnmarshalString(string(b), dynamicResourcesV3)
		if err != nil {
			log.DefaultLogger.Errorf("fail to unmarshal dynamic_resources V3: %v", err)
			return nil, err
		}
		err = dynamicResourcesV3.Validate()
		if err != nil {
			log.DefaultLogger.Errorf("invalid dynamic_resources V3: %v", err)
			return nil, err
		}

		return dynamicResourcesV3, nil
	}

	dynamicResourcesV2 := &envoy_config_bootstrap_v2.Bootstrap_DynamicResources{}
	err = jsonpb.UnmarshalString(string(b), dynamicResourcesV2)
	if err != nil {
		log.DefaultLogger.Errorf("fail to unmarshal dynamic_resources V2: %v", err)
		return nil, err
	}
	err = dynamicResourcesV2.Validate()
	if err != nil {
		log.DefaultLogger.Errorf("invalid dynamic_resources v2: %v", err)
		return nil, err
	}

	return dynamicResourcesV2, nil
}
