package entutils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCallbacksRollbackToAncestor(t *testing.T) {
	// Given callbacks registered before, within, and below a named savepoint.
	var callbacks txCallbacks
	var called []string
	register := func(name string) {
		callbacks.callbacks = append(callbacks.callbacks, func() { called = append(called, name) })
	}
	parent := txSavepointNone.Next()
	child := parent.Next()
	register("outer")
	callbacks.SavePoint(parent)
	register("parent")
	callbacks.SavePoint(child)
	register("child")

	// When rolling back directly to the parent, its checkpoint remains reusable.
	callbacks.RollbackTo(parent)
	register("retry")
	callbacks.RollbackTo(parent)
	register("retained")
	callbacks.Release(parent)

	// Then release preserves callbacks that survived both rollbacks.
	for _, callback := range callbacks.callbacks {
		callback()
	}
	require.Equal(t, []string{"outer", "retained"}, called)
}
