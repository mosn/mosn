package context

import "context"

func Get(ctx context.Context, key ContextKey) interface{} {
	if mosnCtx, ok := ctx.(*valueCtx); ok {
		return mosnCtx.builtin[key]
	}
	return ctx.Value(key)
}

// WithValue add the given key-value pair into the existed value context, or create a new value context contains the pair.
// This Function should not be used along the official context.WithValue !!
// The following context topology will leads to existed pair {'foo':'bar'} NOT FOUND, cause recursive lookup for
// key-type=ContextKey is not supported by mosn.valueCtx.
//
// topology: context.Background -> mosn.valueCtx{'foo':'bar'} -> context.valueCtx -> mosn.valueCtx{'hmm':'haa'}
func WithValue(parent context.Context, key ContextKey, value interface{}) context.Context {
	if mosnCtx, ok := parent.(*valueCtx); ok {
		mosnCtx.builtin[key] = value
		return mosnCtx
	}

	// create new valueCtx
	mosnCtx := &valueCtx{Context: parent}
	mosnCtx.builtin[key] = value
	return mosnCtx
}

// Clone copy the origin mosn value context(if it is), and return new one
func Clone(parent context.Context) context.Context {
	if mosnCtx, ok := parent.(*valueCtx); ok {
		clone := &valueCtx{Context: mosnCtx}
		// array copy assign
		clone.builtin = mosnCtx.builtin
		return clone
	}
	return parent
}
