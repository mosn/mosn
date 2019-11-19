package context

import "context"

type valueCtx struct {
	context.Context

	builtin [ContextKeyEnd]interface{}

	// TODO
	//variables map[string]Variable
}

func (c *valueCtx) Value(key interface{}) interface{} {
	if contextKey, ok := key.(ContextKey); ok {
		return c.builtin[contextKey]
	}
	return c.Context.Value(key)
}
