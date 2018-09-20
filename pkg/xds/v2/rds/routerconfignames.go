package rds

import (
	"sync"
)

var(
	mu          sync.Mutex
	routerNames map[string]bool
)

// AppendRouterName use to append rds router configname to subscript
func AppendRouterName(name string) {
	mu.Lock()
	defer mu.Unlock()
	if routerNames == nil {
		routerNames = make(map[string]bool)
	}
	routerNames[name] = true
}

// GetRouterNames return disctict router config names
func GetRouterNames() []string {
	mu.Lock()
	defer mu.Unlock()
	names := make([]string, len(routerNames))
	i := 0
	for name, _ := range routerNames {
		names[i] = name
		i++
	}
	return names
}
