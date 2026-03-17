package statelessx

import "context"

func BoolFn(fn func() bool) func(context.Context, ...any) bool {
	return func(context.Context, ...any) bool {
		return fn()
	}
}

func Not(fn func() bool) func() bool {
	return func() bool {
		return !fn()
	}
}
