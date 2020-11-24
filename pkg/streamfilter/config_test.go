package streamfilter

import (
	"errors"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"testing"
)

func TestGetStreamFilters(t *testing.T) {
	api.RegisterStream("test1", func(cfg map[string]interface{}) (api.StreamFilterChainFactory, error) {
		return &struct {
			api.StreamFilterChainFactory
		}{}, nil
	})
	api.RegisterStream("test_nil", func(cfg map[string]interface{}) (api.StreamFilterChainFactory, error) {
		return nil, nil
	})
	api.RegisterStream("test_error", func(cfg map[string]interface{}) (api.StreamFilterChainFactory, error) {
		return nil, errors.New("invalid factory create")
	})
	facs := GetStreamFilters([]v2.Filter{
		{Type: "test1"},
		{Type: "test_error"},
		{Type: "not registered"},
		{Type: "test_nil"},
	})
	if len(facs) != 1 {
		t.Fatalf("expected got only one success factory, but got %d", len(facs))
	}
}
