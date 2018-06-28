package router

import (
	"testing"

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func TestRoutersManager_AddRouterName(t *testing.T) {

	routerName := []string{"test1", "test2"}
	RoutersManager.AddRouterNameInList(routerName)

}

func TestRoutersManager_DeleteRouterName(t *testing.T) {
	routerName := []string{"test1", "test2"}
	RoutersManager.AddRouterNameInList(routerName)
	RoutersManager.DeleteRouterNameInList([]string{"test2"})
}

func Test_routersManager_AddRoutersSet(t *testing.T) {
	type args struct {
		routers types.Routers
	}
	tests := []struct {
		name string
		rm   *routersManager
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rm.AddRoutersSet(tt.args.routers)
		})
	}
}

func Test_routersManager_RemoveRouterInRouters(t *testing.T) {
	type args struct {
		routerNames []string
	}
	tests := []struct {
		name string
		rm   *routersManager
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rm.RemoveRouterInRouters(tt.args.routerNames)
		})
	}
}

func Test_routersManager_AddRouterInRouters(t *testing.T) {
	type args struct {
		routerNames []string
	}
	tests := []struct {
		name string
		rm   *routersManager
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rm.AddRouterInRouters(tt.args.routerNames)
		})
	}
}

func Test_routersManager_AddRouterNameInList(t *testing.T) {
	type args struct {
		routerNames []string
	}
	tests := []struct {
		name string
		rm   *routersManager
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rm.AddRouterNameInList(tt.args.routerNames)
		})
	}
}

func Test_routersManager_DeleteRouterNameInList(t *testing.T) {
	type args struct {
		rNames []string
	}
	tests := []struct {
		name string
		rm   *routersManager
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rm.DeleteRouterNameInList(tt.args.rNames)
		})
	}
}
