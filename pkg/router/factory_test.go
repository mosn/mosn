package router

import (
	"reflect"
	"testing"

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func TestRegisteRouterConfigFactory(t *testing.T) {
	type args struct {
		port    types.Protocol
		factory configFactory
	}
	tests := []struct {
		name string
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisteRouterConfigFactory(tt.args.port, tt.args.factory)
		})
	}
}

func TestCreateRouteConfig(t *testing.T) {
	type args struct {
		port   types.Protocol
		config interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    types.Routers
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateRouteConfig(tt.args.port, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateRouteConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateRouteConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
