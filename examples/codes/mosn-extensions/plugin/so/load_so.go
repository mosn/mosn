package main

import (
	"encoding/json"
	"errors"
	"plugin"

	"mosn.io/api"
)

// we consider move it to mosn/pkg later
func init() {
	api.RegisterStream("loadso", CreateLoadSoFilterFactory)
}

type LoadSoConfig struct {
	SoPath string                 `json:"so_path"`
	Config map[string]interface{} `json:"config"`
}

var ErrNotFunc = errors.New("CreateFilterFactory in so is not a factory function")

// CreateLoadSoFilterFactory returns the real factory that implemented by so
func CreateLoadSoFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	b, _ := json.Marshal(conf)
	cfg := &LoadSoConfig{}
	if err := json.Unmarshal(b, cfg); err != nil {
		return nil, err
	}
	p, err := plugin.Open(cfg.SoPath)
	if err != nil {
		return nil, err
	}
	f, err := p.Lookup("CreateFilterFactory")
	if err != nil {
		return nil, err
	}
	function, ok := f.(func(conf map[string]interface{}) (api.StreamFilterChainFactory, error))
	if !ok {
		return nil, ErrNotFunc
	}
	return function(cfg.Config)
}
