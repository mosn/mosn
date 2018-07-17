package router
//
//import (
//	"reflect"
//	"testing"
//
//	"github.com/alipay/sofa-mosn/pkg/api/v2"
//)
//
//func TestNewRouteMatcher(t *testing.T) {
//
//	VHName := "ExampleVH"
//
//	mockProxyAll := &v2.Proxy{
//		VirtualHosts:[]*v2.VirtualHost{
//			{
//				Name:VHName,
//				Domains:[]string{"*","test","*wildcard"},
//				Routers:[]v2.Router{},
//			},
//		},
//	}
//
//	type args struct {
//		config interface{}
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    string
//		wantErr bool
//	}{
//		{
//			name:"testDefault",
//			args:args{
//				config:mockProxyAll,
//			},
//			want:VHName,    // default virtual host's name
//		},
//
//		{
//			name:"testWildcard",
//			args:args{
//				config:mockProxyAll,
//			},
//			want:"wildcard",   // key in tier-2
//		},
//
//		{
//			name:"testVH",
//			args:args{
//				config:mockProxyAll,
//			},
//			want:"test",    // key
//		},
//
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := NewRouteMatcher(tt.args.config)
//
//			rm,_ := got.(*RouteMatcher)
//
//
//			if (err != nil) != tt.wantErr {
//				t.Errorf("NewRouteMatcher() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//
//			if !reflect.DeepEqual(rm.defaultVirtualHost.Name(), tt.want) {
//				t.Errorf("NewRouteMatcher() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
