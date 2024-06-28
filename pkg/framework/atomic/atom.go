package atomic

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
)

// An atom represents a single atomic operation.
type Atom interface {
	context.Context

	// Cancel cancels the atom with the given error.
	Cancel(err ...error)

	// Closes the atom with success.
	succeed()

	// Returns whether the atom is already being run.
	running() bool

	// After calling run, running() is guaranteed to return true. If the atom already finished, it will return an error.
	run() error
}

type atom struct {
	context.Context

	onCancel  sync.Map
	onSuccess sync.Map

	// m protects all below fields.
	m sync.Mutex

	cancel    context.CancelFunc
	err       error
	done      bool
	isRunning bool
}

func RegisterOnCancel(a Atom, fn func()) {
	if a, ok := a.(*atom); ok {
		a.onCancel.Store(ulid.Make(), fn)
	}
}

func RegisterOnSuccess(a Atom, fn func()) {
	if a, ok := a.(*atom); ok {
		a.onSuccess.Store(ulid.Make(), fn)
	}
}

// FromContext returns the Atom from the context.
// If the context is already an Atom, doesn't create a new one.
func FromContext(ctx context.Context) Atom {
	// we cannot wrap atoms as we can't cancel parents
	if a, ok := ctx.(*atom); ok {
		return a
	}

	return NewAtom(ctx)
}

// NewAtom always creates a new Atom from the context.
func NewAtom(ctx context.Context) Atom {
	cCtx, cancel := context.WithCancel(ctx)

	a := &atom{
		Context: cCtx,
		cancel:  cancel,
	}

	return a
}

func (a *atom) Cancel(err ...error) {
	a.m.Lock()
	defer a.m.Unlock()

	// we can't cancel twice, and we can't cancel a success
	if a.Context.Err() != nil || a.done {
		return
	}

	if len(err) > 0 {
		a.err = err[0]
	}

	a.onCancel.Range(func(key, value interface{}) bool {
		if fn, ok := value.(func()); ok {
			fn()
		}
		return true
	})

	a.cancel()
	a.done = true
}

func (a *atom) Err() error {
	a.m.Lock()
	defer a.m.Unlock()

	cErr := a.Context.Err()
	if cErr == nil {
		return nil
	}

	if a.err != nil {
		return a.err
	}

	return cErr
}

// stops future cancels
func (a *atom) succeed() {
	a.m.Lock()
	defer a.m.Unlock()

	a.onSuccess.Range(func(key, value interface{}) bool {
		if fn, ok := value.(func()); ok {
			fn()
		}
		return true
	})

	a.done = true
	a.Context = context.WithoutCancel(a.Context)
}

func (a *atom) running() bool {
	a.m.Lock()
	defer a.m.Unlock()

	return a.isRunning
}

func (a *atom) run() error {
	a.m.Lock()
	defer a.m.Unlock()

	if a.done {
		return fmt.Errorf("atom already done")
	}

	a.isRunning = true
	return nil
}

// RunAtom creates an Atom for the underlying operation.
// The operation can manage the atom as it chooses, and when the operation completes (without panicking) the atom will succeed (regardless of the operation's return value).
func InAtom[Res any](ctx context.Context, op func(a Atom) (Res, error)) (Res, error) {
	atom := FromContext(ctx)

	// we can only succeed the atom if we're the first ones to run it
	canCloseAtom := !atom.running()

	var defRes Res

	err := atom.run()
	if err != nil {
		return defRes, err
	}

	defer func() {
		// we should cancel the associated context but only if we can
		if canCloseAtom {
			defer atom.Cancel()
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			pMsg := stackInRecovery(r)

			// cancel the atom if we panic
			atom.Cancel(fmt.Errorf("panic: %s", pMsg))
			panic(pMsg)
		}
	}()

	type result struct {
		res Res
		err error
	}
	ch := make(chan result)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				pMsg := stackInRecovery(r)

				// cancel the atom if we panic
				atom.Cancel(fmt.Errorf("panic: %s", pMsg))
				panic(pMsg)
			}
		}()
		res, err := op(atom)
		ch <- result{res, err}
	}()

	select {
	case <-atom.Done():
		return defRes, atom.Err()
	case r := <-ch:
		if canCloseAtom {
			atom.succeed()
		}
		return r.res, r.err
	}
}

// AsAtomic wraps an operation to be atomic.
// If the operation returns an error, the atom is cancelled.
func AsAtomic[Req any, Res any](ctx context.Context, op operation.Operation[Req, Res]) operation.Operation[Req, Res] {
	return func(ctx context.Context, req Req) (Res, error) {
		atom := FromContext(ctx)
		defer func() {
			if r := recover(); r != nil {
				pMsg := fmt.Sprintf("%v:\n%s", r, debug.Stack())

				// cancel the atom if we panic
				atom.Cancel(fmt.Errorf("panic: %s", pMsg))
				panic(pMsg)
			}
		}()

		res, err := op(atom, req)
		if err != nil {
			atom.Cancel(err)
		}

		return res, err
	}
}

func stackInRecovery(r interface{}) string {
	return fmt.Sprintf("%v:\n%s", r, debug.Stack())
}
