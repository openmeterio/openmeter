package transaction

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type testContextKey struct{}

func TestRunInNewTransactionShadowsParentTransaction(t *testing.T) {
	parent := context.WithValue(t.Context(), testContextKey{}, "kept")
	parent, err := SetDriverOnContext(parent, noopDriver{})
	require.NoError(t, err)

	createdDriver := &noopDriver{}
	creator := &noopCreator{driver: createdDriver}

	_, err = RunInNewTransaction(parent, creator, func(ctx context.Context) (interface{}, error) {
		require.Equal(t, "kept", ctx.Value(testContextKey{}))

		driver, err := GetDriverFromContext(ctx)
		require.NoError(t, err)
		require.Same(t, createdDriver, driver)

		return nil, nil
	})
	require.NoError(t, err)
	require.True(t, creator.called)

	driver, err := GetDriverFromContext(parent)
	require.NoError(t, err)
	require.NotEqual(t, createdDriver, driver)
}

type noopDriver struct{}

func (noopDriver) Commit() error {
	return nil
}

func (noopDriver) Rollback() error {
	return nil
}

func (noopDriver) SavePoint() error {
	return nil
}

type noopCreator struct {
	called bool
	driver Driver
}

func (n *noopCreator) Tx(ctx context.Context) (context.Context, Driver, error) {
	n.called = true

	return ctx, n.driver, nil
}
