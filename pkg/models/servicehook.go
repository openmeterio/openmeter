package models

import (
	"context"
	"fmt"
	"sync"
)

type ServiceHook[T any] interface {
	PreUpdate(context.Context, *T) error
	PreDelete(context.Context, *T) error
	PostCreate(context.Context, *T) error
	PostUpdate(context.Context, *T) error
	PostDelete(context.Context, *T) error
}

type ServiceHooks[T any] interface {
	RegisterHooks(...ServiceHook[T])
}

var (
	_ ServiceHooks[any] = (*ServiceHookRegistry[any])(nil)
	_ ServiceHook[any]  = (*ServiceHookRegistry[any])(nil)
)

type loopKey string

var loopVal = struct{}{}

type ServiceHookRegistry[T any] struct {
	hooks []ServiceHook[T]

	mu sync.RWMutex

	id loopKey

	once sync.Once
}

func (r *ServiceHookRegistry[T]) init() {
	r.once.Do(func() {
		r.id = loopKey(fmt.Sprintf("service-hook-registry-%p", r))
	})
}

func (r *ServiceHookRegistry[T]) PreUpdate(ctx context.Context, t *T) error {
	r.init()

	if v := ctx.Value(r.id); v != nil {
		return nil
	}

	ctx = context.WithValue(ctx, r.id, loopVal)

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, hook := range r.hooks {
		if err := hook.PreUpdate(ctx, t); err != nil {
			return err
		}
	}

	return nil
}

func (r *ServiceHookRegistry[T]) PreDelete(ctx context.Context, t *T) error {
	r.init()

	if v := ctx.Value(r.id); v != nil {
		return nil
	}

	ctx = context.WithValue(ctx, r.id, loopVal)

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, hook := range r.hooks {
		if err := hook.PreDelete(ctx, t); err != nil {
			return err
		}
	}

	return nil
}

func (r *ServiceHookRegistry[T]) PostCreate(ctx context.Context, t *T) error {
	r.init()

	if v := ctx.Value(r.id); v != nil {
		return nil
	}

	ctx = context.WithValue(ctx, r.id, loopVal)

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, hook := range r.hooks {
		if err := hook.PostCreate(ctx, t); err != nil {
			return err
		}
	}

	return nil
}

func (r *ServiceHookRegistry[T]) PostUpdate(ctx context.Context, t *T) error {
	r.init()

	if v := ctx.Value(r.id); v != nil {
		return nil
	}

	ctx = context.WithValue(ctx, r.id, loopVal)

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, hook := range r.hooks {
		if err := hook.PostUpdate(ctx, t); err != nil {
			return err
		}
	}

	return nil
}

func (r *ServiceHookRegistry[T]) PostDelete(ctx context.Context, t *T) error {
	r.init()

	if v := ctx.Value(r.id); v != nil {
		return nil
	}

	ctx = context.WithValue(ctx, r.id, loopVal)

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, hook := range r.hooks {
		if err := hook.PostDelete(ctx, t); err != nil {
			return err
		}
	}

	return nil
}

func (r *ServiceHookRegistry[T]) RegisterHooks(hooks ...ServiceHook[T]) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.hooks = append(r.hooks, hooks...)
}

func NewServiceHookRegistry[T any]() *ServiceHookRegistry[T] {
	return &ServiceHookRegistry[T]{
		hooks: []ServiceHook[T]{},
		mu:    sync.RWMutex{},
	}
}

var _ ServiceHook[any] = (*NoopServiceHook[any])(nil)

type NoopServiceHook[T any] struct{}

func (n NoopServiceHook[T]) PreUpdate(context.Context, *T) error {
	return nil
}

func (n NoopServiceHook[T]) PreDelete(context.Context, *T) error {
	return nil
}

func (n NoopServiceHook[T]) PostCreate(context.Context, *T) error {
	return nil
}

func (n NoopServiceHook[T]) PostUpdate(context.Context, *T) error {
	return nil
}

func (n NoopServiceHook[T]) PostDelete(context.Context, *T) error {
	return nil
}
