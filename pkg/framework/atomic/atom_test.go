package atomic_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/openmeterio/openmeter/pkg/framework/atomic"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/stretchr/testify/assert"
)

func TestHandlers(t *testing.T) {
	t.Run("Should run handlers on cancel", func(t *testing.T) {
		a := atomic.NewAtom(context.Background())
		calledCounter := 0
		atomic.RegisterOnCancel(a, func() {
			calledCounter++
		})
		atomic.RegisterOnCancel(a, func() {
			calledCounter++
		})
		atomic.RegisterOnCancel(a, func() {
			calledCounter++
		})
		atomic.RegisterOnCancel(a, func() {
			calledCounter++
		})

		a.Cancel()
		assert.Equal(t, 4, calledCounter)
	})
}

func TestInAtom(t *testing.T) {
	t.Run("Operation can cancel the atom any time", func(t *testing.T) {
		ctx := context.Background()
		var opCancelErr error
		res, err := atomic.InAtom(ctx, func(atom atomic.Atom) (interface{}, error) {
			opCancelErr = errors.New("cancel from operation")
			atom.Cancel(opCancelErr)
			return nil, nil
		})
		assert.Error(t, err)
		assert.Empty(t, res)
		assert.Equal(t, opCancelErr, err)
	})

	t.Run("We can have nested runs for a single atom and then cancel", func(t *testing.T) {
		ctx := context.Background()
		var opCancelErr error

		var cancelDidRun bool

		res, err := atomic.InAtom(ctx, func(atom atomic.Atom) (interface{}, error) {

			atomic.RegisterOnCancel(atom, func() {
				cancelDidRun = true
			})

			res, err := atomic.InAtom(atom, func(atom atomic.Atom) (interface{}, error) {
				return 42, nil
			})
			assert.NoError(t, err)
			assert.Equal(t, 42, res)

			opCancelErr = errors.New("cancel from operation")
			atom.Cancel(opCancelErr)

			return res, err
		})
		assert.Error(t, err)
		assert.Empty(t, res)
		assert.Equal(t, opCancelErr, err)
		assert.True(t, cancelDidRun)
	})

	t.Run("We stop sideeffects after cancellation", func(t *testing.T) {
		ctx := context.Background()

		var sideEffectThatIgnoresContext bool
		var sideEffectThatRespectsContext bool

		sideEffectRespectinContext := func(ctx context.Context) {
			if ctx.Err() != nil {
				sideEffectThatRespectsContext = true
			}
		}
		sideEffectIgnoringContext := func() {
			sideEffectThatIgnoresContext = true
		}

		res, err := atomic.InAtom(ctx, func(atom atomic.Atom) (interface{}, error) {
			res, err := atomic.InAtom(atom, func(atom atomic.Atom) (interface{}, error) {
				atom.Cancel()
				return nil, nil
			})

			// ignores the context
			sideEffectIgnoringContext()
			// respects the context
			sideEffectRespectinContext(ctx)

			return res, err
		})

		assert.Error(t, err)
		assert.Empty(t, res)
		assert.False(t, sideEffectThatIgnoresContext)
		assert.False(t, sideEffectThatRespectsContext)
	})

	t.Run("We can have nested runs for a single atom all executing", func(t *testing.T) {
		ctx := context.Background()

		var successDidRun bool
		var cancelDidRun bool

		res, err := atomic.InAtom(ctx, func(atom atomic.Atom) (interface{}, error) {

			atomic.RegisterOnSuccess(atom, func() {
				successDidRun = true
			})

			atomic.RegisterOnCancel(atom, func() {
				cancelDidRun = false
			})

			res, err := atomic.InAtom(atom, func(atom atomic.Atom) (interface{}, error) {
				return 42, nil
			})
			assert.NoError(t, err)
			assert.Equal(t, 42, res)

			return res, err
		})
		assert.NoError(t, err)
		assert.Equal(t, 42, res)

		assert.True(t, successDidRun)
		assert.False(t, cancelDidRun)
	})

	t.Run("We can have multiple goroutines using the same atom", func(t *testing.T) {
		ctx := context.Background()

		var successDidRun bool
		var cancelDidRun bool

		res, err := atomic.InAtom(ctx, func(atom atomic.Atom) (interface{}, error) {

			atomic.RegisterOnSuccess(atom, func() {
				successDidRun = true
			})

			atomic.RegisterOnCancel(atom, func() {
				cancelDidRun = true
			})

			wg := sync.WaitGroup{}
			wg.Add(2)

			go func() {
				defer wg.Done()

				atomic.InAtom(atom, func(atom atomic.Atom) (interface{}, error) {
					return 42, nil
				})
			}()

			go func() {
				defer wg.Done()

				atomic.InAtom(atom, func(atom atomic.Atom) (interface{}, error) {
					atom.Cancel()
					return 42, nil
				})
			}()

			wg.Wait()

			return nil, nil
		})
		assert.Error(t, err)
		assert.Empty(t, res)

		assert.False(t, successDidRun)
		assert.True(t, cancelDidRun)
	})

	t.Run("Should error if atom has already completed", func(t *testing.T) {
		ctx := context.Background()

		atom := atomic.NewAtom(ctx)

		res, err := atomic.InAtom(atom, func(atom atomic.Atom) (interface{}, error) {
			return 42, nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 42, res)

		res, err = atomic.InAtom(atom, func(atom atomic.Atom) (interface{}, error) {
			return 42, nil
		})

		assert.Error(t, err)
		assert.Empty(t, res)
	})
}

func TestAsAtomic(t *testing.T) {
	t.Run("Atomic operation can succeed", func(t *testing.T) {
		ctx := context.Background()

		var successDidRun bool
		var cancelDidRun bool

		res, err := atomic.InAtom(ctx, func(atom atomic.Atom) (interface{}, error) {

			atomic.RegisterOnSuccess(atom, func() {
				successDidRun = true
			})

			atomic.RegisterOnCancel(atom, func() {
				cancelDidRun = false
			})

			var op operation.Operation[any, int] = func(ctx context.Context, _ interface{}) (int, error) {
				return 42, nil
			}

			atomicOp := atomic.AsAtomic(atom, op)

			res, err := atomicOp(ctx, nil)

			assert.NoError(t, err)
			assert.Equal(t, 42, res)

			return res, err
		})
		assert.NoError(t, err)
		assert.Equal(t, 42, res)

		assert.True(t, successDidRun)
		assert.False(t, cancelDidRun)
	})

	t.Run("Atomic operation can fail", func(t *testing.T) {
		ctx := context.Background()

		var successDidRun bool
		var cancelDidRun bool

		res, err := atomic.InAtom(ctx, func(atom atomic.Atom) (interface{}, error) {

			atomic.RegisterOnSuccess(atom, func() {
				successDidRun = true
			})

			atomic.RegisterOnCancel(atom, func() {
				cancelDidRun = false
			})

			var op operation.Operation[any, int] = func(ctx context.Context, _ interface{}) (int, error) {
				return 42, errors.New("error")
			}

			atomicOp := atomic.AsAtomic(atom, op)

			res, err := atomicOp(ctx, nil)

			assert.Error(t, err)
			assert.Equal(t, 42, res)

			return res, err
		})

		assert.Error(t, err)
		assert.Equal(t, 42, res)

		assert.False(t, successDidRun)
		assert.True(t, cancelDidRun)
	})
}
