package istio

import "encoding/json"

type adsConfig struct {
	parseConfig func(dynamic, static json.RawMessage) (XdsStreamConfig, error)
}

var globalAds = adsConfig{
	parseConfig: func(dynamic, static json.RawMessage) (XdsStreamConfig, error) {
		// default no xds action for test
		return nil, nil
	},
}

func ParseAdsConfig(dynamic, static json.RawMessage) (XdsStreamConfig, error) {
	return globalAds.parseConfig(dynamic, static)
}

func ResgiterParseAdsConfig(f func(dynamic, static json.RawMessage) (XdsStreamConfig, error)) {
	globalAds.parseConfig = f
}
