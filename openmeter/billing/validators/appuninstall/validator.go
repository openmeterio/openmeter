package appuninstall

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/servicehooks"
)

// HookName identifies the billing app-usage validator in the app lifecycle registry.
const HookName = "billing.app-usage"

type appUsageChecker interface {
	IsAppUsed(ctx context.Context, appID app.AppID) error
}

var _ servicehooks.Hook[app.LifecycleEvent] = (*Validator)(nil)

type Validator struct {
	appUsageChecker appUsageChecker
}

func NewValidator(appUsageChecker appUsageChecker) (*Validator, error) {
	if appUsageChecker == nil {
		return nil, errors.New("app usage checker is required")
	}

	return &Validator{
		appUsageChecker: appUsageChecker,
	}, nil
}

func (v *Validator) Handle(ctx context.Context, event app.LifecycleEvent) error {
	if event.Operation != app.OperationKindUninstall {
		return nil
	}

	if event.Before == nil {
		return errors.New("app state before uninstall is required")
	}

	if err := v.appUsageChecker.IsAppUsed(ctx, event.Before.GetID()); err != nil {
		return fmt.Errorf("validating billing app usage: %w", err)
	}

	return nil
}
