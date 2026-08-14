package appuninstall

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
)

type appUsageCheckerStub struct {
	appID app.AppID
	err   error
	calls int
}

func (s *appUsageCheckerStub) IsAppUsed(_ context.Context, appID app.AppID) error {
	s.appID = appID
	s.calls++

	return s.err
}

func TestValidatorHandlesUninstallOperation(t *testing.T) {
	rejected := errors.New("app is in use")
	checker := &appUsageCheckerStub{err: rejected}
	validator, err := NewValidator(checker)
	require.NoError(t, err)

	before := app.AppBase{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: "test"},
			ID:              "app-1",
		},
	}

	err = validator.Handle(t.Context(), app.LifecycleEvent{
		Operation: app.OperationKindUninstall,
		Before:    &before,
	})

	require.ErrorIs(t, err, rejected)
	require.Equal(t, 1, checker.calls)
	require.Equal(t, before.GetID(), checker.appID)
}

func TestValidatorIgnoresOtherOperations(t *testing.T) {
	checker := &appUsageCheckerStub{}
	validator, err := NewValidator(checker)
	require.NoError(t, err)

	err = validator.Handle(t.Context(), app.LifecycleEvent{
		Operation: app.OperationKind("create"),
	})

	require.NoError(t, err)
	require.Zero(t, checker.calls)
}

func TestValidatorRequiresStateBeforeUninstall(t *testing.T) {
	checker := &appUsageCheckerStub{}
	validator, err := NewValidator(checker)
	require.NoError(t, err)

	err = validator.Handle(t.Context(), app.LifecycleEvent{
		Operation: app.OperationKindUninstall,
	})

	require.EqualError(t, err, "app state before uninstall is required")
	require.Zero(t, checker.calls)
}
