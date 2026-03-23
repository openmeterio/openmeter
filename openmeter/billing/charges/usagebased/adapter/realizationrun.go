package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	dbchargeusagebasedruns "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedruns"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ usagebased.RealizationRunAdapter = (*adapter)(nil)

func (a *adapter) CreateRealizationRun(ctx context.Context, chargeID meta.ChargeID, input usagebased.CreateRealizationRunInput) (usagebased.RealizationRunBase, error) {
	if err := chargeID.Validate(); err != nil {
		return usagebased.RealizationRunBase{}, err
	}

	if err := input.Validate(); err != nil {
		return usagebased.RealizationRunBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.RealizationRunBase, error) {
		create := tx.db.ChargeUsageBasedRuns.Create().
			SetNamespace(chargeID.Namespace).
			SetChargeID(chargeID.ID).
			SetType(input.Type).
			SetAsof(input.AsOf.UTC()).
			SetCollectionEnd(input.CollectionEnd.UTC()).
			SetMeterValue(input.MeterValue)

		create = totals.Set(create, input.Totals)

		dbRun, err := create.Save(ctx)
		if err != nil {
			return usagebased.RealizationRunBase{}, err
		}

		return MapRealizationRunBaseFromDB(dbRun), nil
	})
}

func (a *adapter) UpdateRealizationRun(ctx context.Context, input usagebased.UpdateRealizationRunInput) (usagebased.RealizationRunBase, error) {
	if err := input.Validate(); err != nil {
		return usagebased.RealizationRunBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.RealizationRunBase, error) {
		update := tx.db.ChargeUsageBasedRuns.UpdateOneID(input.ID.ID).
			Where(dbchargeusagebasedruns.NamespaceEQ(input.ID.Namespace)).
			SetAsof(input.AsOf.UTC()).
			SetMeterValue(input.MeterValue)

		update = totals.Set(update, input.Totals)

		dbRun, err := update.Save(ctx)
		if err != nil {
			return usagebased.RealizationRunBase{}, err
		}

		return MapRealizationRunBaseFromDB(dbRun), nil
	})
}
