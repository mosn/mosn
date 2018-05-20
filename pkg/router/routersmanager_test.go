package router

import (
	"testing"
)

func TestRoutersManager_AddRouterName(t *testing.T){
	
	routerName := []string{"test1","test2"}
	RoutersManager.AddRouterNameInList(routerName)
	
}

func TestRoutersManager_DeleteRouterName(t *testing.T) {
	routerName := []string{"test1", "test2"}
	RoutersManager.AddRouterNameInList(routerName)
	RoutersManager.DeleteRouterNameInList([]string{"test2"})
}
