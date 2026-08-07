package servicehooks

import (
	"cmp"
	"context"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Hook handles one service event. The event type and its lifecycle semantics
// belong to the service registering the hook.
type Hook[T any] interface {
	Handle(context.Context, T) error
}

// HookFunc adapts a function to Hook.
type HookFunc[T any] func(context.Context, T) error

func (f HookFunc[T]) Handle(ctx context.Context, event T) error {
	return f(ctx, event)
}

// hookToken is intentionally non-zero-sized so distinct registrations always
// have distinct pointer identities.
type hookToken struct {
	_ byte
}

type registration[T any] struct {
	name        string
	hook        Hook[T]
	priority    Priority
	cyclePolicy CyclePolicy
	token       *hookToken
}

// Registry invokes typed hooks in priority order. Its zero value is ready for
// use. A registry is sealed explicitly with Seal or automatically by its first
// invocation; registrations after sealing fail with ErrRegistrySealed.
//
// Registry must not be copied after first use.
type Registry[T any] struct {
	mu sync.RWMutex

	registrations []registration[T]
	names         map[string]struct{}
	sealed        bool
}

// NewRegistry creates an empty registry.
func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{}
}

// Register adds a named hook. Names must be unique within a registry. Hooks
// with lower priorities run first; equal priorities preserve registration
// order. PriorityDefault is used when WithPriority is absent.
func (r *Registry[T]) Register(name string, hook Hook[T], options ...RegisterOption) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrHookNameRequired
	}

	if isNilHook(hook) {
		return ErrHookRequired
	}

	config := defaultRegisterConfig()
	for i, option := range options {
		if option == nil {
			return &registerOptionError{Index: i, Err: ErrRegisterOptionInvalid}
		}

		if err := option(&config); err != nil {
			return &registerOptionError{Index: i, Err: err}
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.sealed {
		return ErrRegistrySealed
	}

	if _, ok := r.names[name]; ok {
		return &duplicateHookError{Name: name}
	}

	if r.names == nil {
		r.names = make(map[string]struct{})
	}

	r.names[name] = struct{}{}
	r.registrations = append(r.registrations, registration[T]{
		name:        name,
		hook:        hook,
		priority:    config.priority,
		cyclePolicy: config.cyclePolicy,
		token:       &hookToken{},
	})

	slices.SortStableFunc(r.registrations, func(a, b registration[T]) int {
		return cmp.Compare(a.priority, b.priority)
	})

	return nil
}

// Seal prevents further registration. Invoke seals the registry automatically.
func (r *Registry[T]) Seal() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sealed = true
}

// IsSealed reports whether registration has ended.
func (r *Registry[T]) IsSealed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.sealed
}

// Invoke calls registered hooks synchronously and stops at the first error.
// Hook callbacks run without holding the registry lock. Concurrent invocations
// are allowed, so hook implementations must provide their own synchronization.
func (r *Registry[T]) Invoke(ctx context.Context, event T) error {
	if ctx == nil {
		return ErrContextRequired
	}

	registrations := r.sealAndGetRegistrations()
	if err := ctx.Err(); err != nil {
		return err
	}

	active := activeHookFromContext(ctx)
	for _, registered := range registrations {
		if !isHookActive(active, registered.token) {
			continue
		}

		if registered.cyclePolicy == CyclePolicyError {
			return CycleError{
				HookName: registered.name,
				Priority: registered.priority,
			}
		}
	}

	for _, registered := range registrations {
		if isHookActive(active, registered.token) {
			continue
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		if err := invokeRegisteredHook(ctx, event, active, registered); err != nil {
			return InvocationError{
				HookName: registered.name,
				Priority: registered.priority,
				Err:      err,
			}
		}
	}

	return nil
}

func invokeRegisteredHook[T any](ctx context.Context, event T, active *activeHook, registered registration[T]) error {
	frame := &activeHook{
		parent: active,
		token:  registered.token,
	}
	frame.running.Store(true)
	defer frame.running.Store(false)

	hookCtx := context.WithValue(ctx, activeHookContextKey{}, frame)

	return registered.hook.Handle(hookCtx, event)
}

func (r *Registry[T]) sealAndGetRegistrations() []registration[T] {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sealed = true

	return r.registrations
}

func isNilHook[T any](hook Hook[T]) bool {
	if hook == nil {
		return true
	}

	value := reflect.ValueOf(hook)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

type activeHookContextKey struct{}

type activeHook struct {
	parent  *activeHook
	token   *hookToken
	running atomic.Bool
}

func activeHookFromContext(ctx context.Context) *activeHook {
	active, _ := ctx.Value(activeHookContextKey{}).(*activeHook)

	return active
}

func isHookActive(active *activeHook, token *hookToken) bool {
	for frame := active; frame != nil; frame = frame.parent {
		if frame.token == token && frame.running.Load() {
			return true
		}
	}

	return false
}

type registerOptionError struct {
	Index int
	Err   error
}

func (e registerOptionError) Error() string {
	return "invalid service hook register option[" + strconv.Itoa(e.Index) + "]: " + e.Err.Error()
}

func (e registerOptionError) Unwrap() []error {
	return []error{ErrRegisterOptionInvalid, e.Err}
}

type duplicateHookError struct {
	Name string
}

func (e duplicateHookError) Error() string {
	return "service hook " + e.Name + ": " + ErrHookAlreadyRegistered.Error()
}

func (e duplicateHookError) Unwrap() error {
	return ErrHookAlreadyRegistered
}
