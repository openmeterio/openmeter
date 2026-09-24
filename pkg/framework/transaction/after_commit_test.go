package transaction

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type commitRecordingDriver struct {
	commits   int
	rollbacks int
	savepoint int
	started   bool
}

func (d *commitRecordingDriver) Commit() error {
	if d.savepoint > 0 {
		d.savepoint--
		return nil
	}
	d.commits++
	d.started = false
	return nil
}

func (d *commitRecordingDriver) Rollback() error {
	if d.savepoint > 0 {
		d.savepoint--
		return nil
	}
	d.rollbacks++
	d.started = false
	return nil
}

func (d *commitRecordingDriver) SavePoint() error {
	if !d.started {
		d.started = true
		return nil
	}
	d.savepoint++
	return nil
}

func TestAfterCommitWaitsForOutermostCommit(t *testing.T) {
	driver := &commitRecordingDriver{}
	creator := &noopCreator{driver: driver}
	var published []string

	_, err := Run(t.Context(), creator, func(ctx context.Context) (struct{}, error) {
		require.NoError(t, AfterCommit(ctx, func() { published = append(published, "outer") }))
		_, err := Run(ctx, creator, func(ctx context.Context) (struct{}, error) {
			require.NoError(t, AfterCommit(ctx, func() { published = append(published, "inner") }))
			return struct{}{}, nil
		})
		require.NoError(t, err)
		require.Empty(t, published)
		require.Zero(t, driver.commits)
		return struct{}{}, nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, driver.commits)
	require.Equal(t, []string{"outer", "inner"}, published)
}

func TestAfterCommitDiscardsRolledBackScopes(t *testing.T) {
	driver := &commitRecordingDriver{}
	creator := &noopCreator{driver: driver}
	var published []string

	_, err := Run(t.Context(), creator, func(ctx context.Context) (struct{}, error) {
		_, innerErr := Run(ctx, creator, func(ctx context.Context) (struct{}, error) {
			require.NoError(t, AfterCommit(ctx, func() { published = append(published, "inner") }))
			return struct{}{}, errors.New("roll back inner")
		})
		require.ErrorContains(t, innerErr, "roll back inner")
		require.NoError(t, AfterCommit(ctx, func() { published = append(published, "outer") }))
		return struct{}{}, nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{"outer"}, published)

	_, err = Run(t.Context(), creator, func(ctx context.Context) (struct{}, error) {
		require.NoError(t, AfterCommit(ctx, func() { published = append(published, "rolled back") }))
		return struct{}{}, errors.New("roll back outer")
	})
	require.ErrorContains(t, err, "roll back outer")
	require.Equal(t, []string{"outer"}, published)
}

func TestAfterCommitThreeNestedScopes(t *testing.T) {
	for _, tc := range []struct {
		name           string
		rollbackMiddle bool
		rollbackOuter  bool
		expectPublish  bool
	}{
		{name: "all scopes commit", expectPublish: true},
		{name: "middle scope rolls back", rollbackMiddle: true},
		{name: "outer scope rolls back", rollbackOuter: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given three nested transaction scopes and a callback in the innermost scope
			driver := &commitRecordingDriver{}
			creator := &noopCreator{driver: driver}
			published := false

			// when the innermost scope succeeds and an enclosing scope optionally rolls back
			_, err := Run(t.Context(), creator, func(ctx context.Context) (struct{}, error) {
				_, middleErr := Run(ctx, creator, func(ctx context.Context) (struct{}, error) {
					_, innerErr := Run(ctx, creator, func(ctx context.Context) (struct{}, error) {
						require.NoError(t, AfterCommit(ctx, func() { published = true }))
						return struct{}{}, nil
					})
					require.NoError(t, innerErr)
					require.False(t, published)
					if tc.rollbackMiddle {
						return struct{}{}, errors.New("roll back middle")
					}
					return struct{}{}, nil
				})
				if tc.rollbackMiddle {
					require.ErrorContains(t, middleErr, "roll back middle")
				} else {
					require.NoError(t, middleErr)
				}
				require.False(t, published)
				if tc.rollbackOuter {
					return struct{}{}, errors.New("roll back outer")
				}
				return struct{}{}, nil
			})
			if tc.rollbackOuter {
				require.ErrorContains(t, err, "roll back outer")
			} else {
				require.NoError(t, err)
			}

			// then the callback runs only after the outer commit and only if its scope survived
			require.Equal(t, tc.expectPublish, published)
		})
	}
}

func TestAfterCommitInIndependentTransactionDoesNotWaitForParent(t *testing.T) {
	parent := &noopCreator{driver: &commitRecordingDriver{}}
	independent := &noopCreator{driver: &commitRecordingDriver{}}
	var published []string

	_, err := Run(t.Context(), parent, func(ctx context.Context) (struct{}, error) {
		require.NoError(t, AfterCommit(ctx, func() { published = append(published, "parent") }))
		_, err := RunInNewTransaction(ctx, independent, func(ctx context.Context) (struct{}, error) {
			require.NoError(t, AfterCommit(ctx, func() { published = append(published, "independent") }))
			return struct{}{}, nil
		})
		require.NoError(t, err)
		require.Equal(t, []string{"independent"}, published)
		return struct{}{}, errors.New("roll back parent")
	})
	require.ErrorContains(t, err, "roll back parent")
	require.Equal(t, []string{"independent"}, published)
}
