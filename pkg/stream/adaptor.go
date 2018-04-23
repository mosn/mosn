package stream

import (
	"context"
)

func GetMap(context context.Context, defaultSize int) map[string]interface{} {
	return make(map[string]interface{}, defaultSize)
}
