package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	dbchargeflatfeerun "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeerun"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) createCurrentRun(ctx context.Context, dbCharge *db.ChargeFlatFee, runTotals totals.Totals, noFiatTransactionRequired bool) (*db.ChargeFlatFeeRun, error) {
	runCreate := a.db.ChargeFlatFeeRun.Create().
		SetNamespace(dbCharge.Namespace).
		SetChargeID(dbCharge.ID).
		SetType(flatfee.RealizationRunTypeFinalRealization).
		SetInitialType(flatfee.RealizationRunTypeFinalRealization).
		SetServicePeriodFrom(dbCharge.ServicePeriodFrom).
		SetServicePeriodTo(dbCharge.ServicePeriodTo).
		SetAmountAfterProration(dbCharge.AmountAfterProration).
		SetNoFiatTransactionRequired(noFiatTransactionRequired)

	runCreate = totals.Set(runCreate, runTotals)

	dbRun, err := runCreate.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating flat fee realization run [charge_id=%s]: %w", dbCharge.ID, err)
	}

	if _, err := a.db.ChargeFlatFee.UpdateOneID(dbCharge.ID).
		Where(dbchargeflatfee.NamespaceEQ(dbCharge.Namespace)).
		SetCurrentRealizationRunID(dbRun.ID).
		Save(ctx); err != nil {
		return nil, fmt.Errorf("setting flat fee current run [charge_id=%s, run_id=%s]: %w", dbCharge.ID, dbRun.ID, err)
	}

	return dbRun, nil
}

func (a *adapter) ProvisionCurrentRun(ctx context.Context, input flatfee.ProvisionCurrentRunInput) (flatfee.RealizationRunBase, error) {
	if err := input.Validate(); err != nil {
		return flatfee.RealizationRunBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.RealizationRunBase, error) {
		dbCharge, err := tx.db.ChargeFlatFee.Query().
			Where(
				dbchargeflatfee.NamespaceEQ(input.Charge.Namespace),
				dbchargeflatfee.IDEQ(input.Charge.ID),
			).
			WithCurrentRun().
			Only(ctx)
		if err != nil {
			return flatfee.RealizationRunBase{}, fmt.Errorf("querying flat fee charge [id=%s]: %w", input.Charge.ID, err)
		}

		if dbCharge.CurrentRealizationRunID != nil {
			dbRun, err := dbCharge.Edges.CurrentRunOrErr()
			if err != nil {
				return flatfee.RealizationRunBase{}, fmt.Errorf("querying flat fee current run [charge_id=%s]: %w", input.Charge.ID, err)
			}

			return mapRealizationRunBaseFromDB(dbRun), nil
		}

		dbRun, err := tx.createCurrentRun(ctx, dbCharge, totals.Totals{}, input.NoFiatTransactionRequired)
		if err != nil {
			return flatfee.RealizationRunBase{}, err
		}

		return mapRealizationRunBaseFromDB(dbRun), nil
	})
}

func (a *adapter) currentRunByChargeID(ctx context.Context, chargeID meta.ChargeID) (*db.ChargeFlatFeeRun, error) {
	if err := chargeID.Validate(); err != nil {
		return nil, err
	}

	dbRun, err := queryCurrentRunByChargeID(a.db, chargeID).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying flat fee current run [charge_id=%s]: %w", chargeID.ID, err)
	}

	return dbRun, nil
}

func (a *adapter) updateCurrentRunTotals(ctx context.Context, dbRun *db.ChargeFlatFeeRun, runTotals totals.Totals) (*db.ChargeFlatFeeRun, error) {
	updated, err := totals.Set(
		a.db.ChargeFlatFeeRun.UpdateOneID(dbRun.ID).
			Where(dbchargeflatfeerun.NamespaceEQ(dbRun.Namespace)),
		runTotals,
	).Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("updating flat fee current run totals [run_id=%s]: %w", dbRun.ID, err)
	}

	return updated, nil
}

func queryCurrentRunByChargeID(dbClient *db.Client, chargeID meta.ChargeID) *db.ChargeFlatFeeRunQuery {
	return dbClient.ChargeFlatFee.Query().
		Where(
			dbchargeflatfee.NamespaceEQ(chargeID.Namespace),
			dbchargeflatfee.IDEQ(chargeID.ID),
			dbchargeflatfee.CurrentRealizationRunIDNotNil(),
		).
		QueryCurrentRun()
}
