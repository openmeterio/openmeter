package adapter

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/billing/periodicrealization"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	rundb "github.com/openmeterio/openmeter/openmeter/ent/db/billingperiodicrealizationrun"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) GetRunCovering(ctx context.Context, input periodicrealization.GetOrCreateRunAtInput) (*periodicrealization.Run, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*periodicrealization.Run, error) {
		if err := input.Validate(); err != nil {
			return nil, err
		}

		// Soft-deleted runs count too: they still own their period.
		row, err := tx.db.BillingPeriodicRealizationRun.Query().
			Where(rundb.Namespace(input.Namespace), rundb.PeriodToGTE(input.AsOf.UTC())).
			Order(rundb.ByPeriodTo(sql.OrderAsc())).
			First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, nil
			}

			return nil, fmt.Errorf("get periodic realization run covering a time: %w", err)
		}

		run := fromDBBillingPeriodicRealizationRun(row)

		return &run, nil
	})
}

func (a *adapter) GetLatestRun(ctx context.Context, namespace string) (*periodicrealization.Run, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*periodicrealization.Run, error) {
		if namespace == "" {
			return nil, models.NewGenericValidationError(errors.New("namespace is required"))
		}

		// Soft-deleted runs count too: they still own their period.
		row, err := tx.db.BillingPeriodicRealizationRun.Query().
			Where(rundb.Namespace(namespace)).
			Order(rundb.ByPeriodTo(sql.OrderDesc())).
			First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, nil
			}

			return nil, fmt.Errorf("get latest periodic realization run: %w", err)
		}

		run := fromDBBillingPeriodicRealizationRun(row)

		return &run, nil
	})
}

func (a *adapter) CreateRun(ctx context.Context, input periodicrealization.CreateRunInput) (periodicrealization.Run, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (periodicrealization.Run, error) {
		if err := input.Validate(); err != nil {
			return periodicrealization.Run{}, err
		}

		row, err := tx.db.BillingPeriodicRealizationRun.Create().
			SetNamespace(input.Namespace).
			SetNillablePreviousRunID(input.PreviousRunID).
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
