package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	dbchargeflatfeerun "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeerun"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ flatfee.ChargeRunAdapter = (*adapter)(nil)

func (a *adapter) CreateCurrentRun(ctx context.Context, input flatfee.CreateCurrentRunInput) (flatfee.RealizationRunBase, error) {
	if err := input.Validate(); err != nil {
		return flatfee.RealizationRunBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.RealizationRunBase, error) {
		dbCharge, err := tx.db.ChargeFlatFee.Query().
			Where(
				dbchargeflatfee.NamespaceEQ(input.Charge.Namespace),
				dbchargeflatfee.IDEQ(input.Charge.ID),
			).
			Only(ctx)
		if err != nil {
			return flatfee.RealizationRunBase{}, fmt.Errorf("querying flat fee charge [id=%s]: %w", input.Charge.ID, err)
		}

		if dbCharge.CurrentRealizationRunID != nil {
			return flatfee.RealizationRunBase{}, fmt.Errorf("flat fee charge [id=%s] already has current run [run_id=%s]", input.Charge.ID, *dbCharge.CurrentRealizationRunID)
		}

		runCreate := tx.db.ChargeFlatFeeRun.Create().
			SetNamespace(dbCharge.Namespace).
			SetChargeID(dbCharge.ID).
			SetType(flatfee.RealizationRunTypeFinalRealization).
			SetInitialType(flatfee.RealizationRunTypeFinalRealization).
			SetServicePeriodFrom(input.ServicePeriod.From).
			SetServicePeriodTo(input.ServicePeriod.To).
			SetAmountAfterProration(input.AmountAfterProration).
			SetNoFiatTransactionRequired(input.NoFiatTransactionRequired).
			SetImmutable(input.Immutable).
			SetNillableLineID(input.LineID).
			SetNillableInvoiceID(input.InvoiceID)

		runCreate = totals.Set(runCreate, totals.Totals{})

		dbRun, err := runCreate.Save(ctx)
		if err != nil {
			return flatfee.RealizationRunBase{}, fmt.Errorf("creating current flat fee realization run [charge_id=%s]: %w", dbCharge.ID, err)
		}

		if _, err := tx.db.ChargeFlatFee.UpdateOneID(dbCharge.ID).
			Where(dbchargeflatfee.NamespaceEQ(dbCharge.Namespace)).
			SetCurrentRealizationRunID(dbRun.ID).
			Save(ctx); err != nil {
			return flatfee.RealizationRunBase{}, fmt.Errorf("setting flat fee current run [charge_id=%s, run_id=%s]: %w", dbCharge.ID, dbRun.ID, err)
		}

		return mapRealizationRunBaseFromDB(dbRun), nil
	})
}

func (a *adapter) UpdateRealizationRun(ctx context.Context, input flatfee.UpdateRealizationRunInput) (flatfee.RealizationRunBase, error) {
	input = input.Normalized()

	if err := input.Validate(); err != nil {
		return flatfee.RealizationRunBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.RealizationRunBase, error) {
		update := tx.db.ChargeFlatFeeRun.UpdateOneID(input.ID.ID).
			Where(dbchargeflatfeerun.NamespaceEQ(input.ID.Namespace))

		if input.Type.IsPresent() {
			update = update.SetType(input.Type.OrEmpty())
		}

		if input.DeletedAt.IsPresent() {
			update = update.SetOrClearDeletedAt(input.DeletedAt.OrEmpty())
		}

		if input.LineID.IsPresent() {
			update = update.SetOrClearLineID(input.LineID.OrEmpty())
		}

		if input.InvoiceID.IsPresent() {
			update = update.SetOrClearInvoiceID(input.InvoiceID.OrEmpty())
		}

		if input.ServicePeriod.IsPresent() {
			servicePeriod := input.ServicePeriod.OrEmpty()
			update = update.
				SetServicePeriodFrom(servicePeriod.From).
				SetServicePeriodTo(servicePeriod.To)
		}

		if input.AmountAfterProration.IsPresent() {
			update = update.SetAmountAfterProration(input.AmountAfterProration.OrEmpty())
		}

		if input.Totals.IsPresent() {
			update = totals.Set(update, input.Totals.OrEmpty())
		}

		if input.NoFiatTransactionRequired.IsPresent() {
			update = update.SetNoFiatTransactionRequired(input.NoFiatTransactionRequired.OrEmpty())
		}

		if input.Immutable.IsPresent() {
			update = update.SetImmutable(input.Immutable.OrEmpty())
		}

		dbRun, err := update.Save(ctx)
		if err != nil {
			return flatfee.RealizationRunBase{}, fmt.Errorf("updating flat fee realization run [run_id=%s]: %w", input.ID.ID, err)
		}

		return mapRealizationRunBaseFromDB(dbRun), nil
	})
}

func (a *adapter) DetachCurrentRun(ctx context.Context, chargeID meta.ChargeID) error {
	if err := chargeID.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if _, err := tx.db.ChargeFlatFee.Update().
			Where(
				dbchargeflatfee.NamespaceEQ(chargeID.Namespace),
				dbchargeflatfee.IDEQ(chargeID.ID),
			).
			ClearCurrentRealizationRunID().
			Save(ctx); err != nil {
			return fmt.Errorf("detach flat fee current run [charge_id=%s]: %w", chargeID.ID, err)
		}

		return nil
	})
}
