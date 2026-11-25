package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/periodicrealization"
	rundb "github.com/openmeterio/openmeter/openmeter/ent/db/billingperiodicrealizationrun"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) HasRuns(ctx context.Context, namespace string) (bool, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (bool, error) {
		if namespace == "" {
			return false, models.NewGenericValidationError(errors.New("namespace is required"))
		}

		// Soft-deleted runs count too: they still own their period.
		exists, err := tx.db.BillingPeriodicRealizationRun.Query().
			Where(rundb.Namespace(namespace)).
			Exist(ctx)
		if err != nil {
			return false, fmt.Errorf("check periodic realization runs: %w", err)
		}

		return exists, nil
	})
}

func (a *adapter) CreateRun(ctx context.Context, input periodicrealization.CreateRunInput) (periodicrealization.Run, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (periodicrealization.Run, error) {
		if err := input.Validate(); err != nil {
			return periodicrealization.Run{}, err
		}

		row, err := tx.db.BillingPeriodicRealizationRun.Create().
			SetNamespace(input.Namespace).
			SetPeriodTo(input.PeriodTo.UTC()).
			SetRealizeAfter(input.RealizeAfter.UTC()).
			SetStatus(periodicrealization.RunStatusScheduled).
			Save(ctx)
		if err != nil {
			return periodicrealization.Run{}, fmt.Errorf("create periodic realization run: %w", err)
		}

		return fromDBBillingPeriodicRealizationRun(row), nil
	})
}
