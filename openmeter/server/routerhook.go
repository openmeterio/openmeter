package server

import (
	"net/http"
	"slices"

	"github.com/samber/lo"
)

type RouteManager interface {
	Mount(pattern string, h http.Handler)
	Handle(pattern string, h http.Handler)
	HandleFunc(pattern string, h http.HandlerFunc)
	Method(method, pattern string, h http.Handler)
	MethodFunc(method, pattern string, h http.HandlerFunc)
	Connect(pattern string, h http.HandlerFunc)
	Delete(pattern string, h http.HandlerFunc)
	Get(pattern string, h http.HandlerFunc)
	Head(pattern string, h http.HandlerFunc)
	Options(pattern string, h http.HandlerFunc)
	Patch(pattern string, h http.HandlerFunc)
	Post(pattern string, h http.HandlerFunc)
	Put(pattern string, h http.HandlerFunc)
	Trace(pattern string, h http.HandlerFunc)
}

type MiddlewareManager interface {
	Use(middlewares ...Middleware)
}

type Middleware = func(http.Handler) http.Handler

type MiddlewareHook func(m MiddlewareManager)

type RouteHook func(r RouteManager)

type RouterHookManager interface {
	RegisterMiddleware(priority int, middleware Middleware)
	RegisterMiddlewareHook(priority int, mHook MiddlewareHook)
	RegisterRouteHook(priority int, route RouteHook)
	Apply(impl interface {
		MiddlewareManager
		RouteManager
	})
}

func NewRouterHookManager() RouterHookManager {
	return &routerHooks{
		middlewares: make(map[int][]MiddlewareHook),
		routes:      make(map[int][]RouteHook),
	}
}

// Alternatively we could manage it with DI:
// - if a given middleware needs to be registered after another one, it could require it in DI, and part of DI is registering the handlers...
type routerHooks struct {
	middlewares map[int][]MiddlewareHook
	routes      map[int][]RouteHook
}

// Lower priority is registered first
func (r *routerHooks) RegisterMiddleware(priority int, middleware Middleware) {
	hooksWithPriority, ok := r.middlewares[priority]
	if !ok {
		hooksWithPriority = make([]MiddlewareHook, 0, 1)
	}

	r.middlewares[priority] = append(hooksWithPriority, func(m MiddlewareManager) {
		m.Use(middleware)
	})
}

// Lower priority is registered first
func (r *routerHooks) RegisterMiddlewareHook(priority int, mHook MiddlewareHook) {
	hooksWithPriority, ok := r.middlewares[priority]
	if !ok {
		hooksWithPriority = make([]MiddlewareHook, 0, 1)
	}

	r.middlewares[priority] = append(hooksWithPriority, mHook)
}

// Lower priority is registered first
func (r *routerHooks) RegisterRouteHook(priority int, route RouteHook) {
	hooksWithPriority, ok := r.routes[priority]
	if !ok {
		hooksWithPriority = make([]RouteHook, 0, 1)
	}

	r.routes[priority] = append(hooksWithPriority, route)
}

func (r *routerHooks) Apply(impl interface {
	MiddlewareManager
	RouteManager
},
) {
	mKeys := lo.Keys(r.middlewares)
	slices.Sort(mKeys)
	for _, key := range mKeys {
		hooks, ok := r.middlewares[key]
		if !ok {
			continue
		}

		for _, mHook := range hooks {
			mHook(impl)
		}
	}

	rKeys := lo.Keys(r.routes)
	slices.Sort(rKeys)
	for _, key := range rKeys {
		hooks, ok := r.routes[key]
		if !ok {
			continue
		}

		for _, rHook := range hooks {
			rHook(impl)
		}
	}
}
