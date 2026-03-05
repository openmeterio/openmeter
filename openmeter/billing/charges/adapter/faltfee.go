package adapter

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) UpdateFlatFeeCharge(ctx context.Context, charge charges.FlatFeeCharge) (charges.FlatFeeCharge, error) {
	if err := charge.ManagedModel.Validate(); err != nil {
		return charges.FlatFeeCharge{}, err
	}

	if err := charge.Validate(); err != nil {
		return charges.FlatFeeCharge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.FlatFeeCharge, error) {
		intent := charge.Intent

		dbEntity, err := tx.updateChargeIntent(ctx, charge.GetChargeID(), intent.IntentMeta, charge.Status)
		if err != nil {
			return charges.FlatFeeCharge{}, err
		}

		var discounts *productcatalog.Discounts
		if intent.PercentageDiscounts != nil {
			discounts = &productcatalog.Discounts{Percentage: intent.PercentageDiscounts}
		}

		proRating, err := proRatingConfigToDB(intent.ProRating)
		if err != nil {
			return charges.FlatFeeCharge{}, err
		}

		create := tx.db.ChargeFlatFee.UpdateOneID(charge.ID).
			Where(dbchargeflatfee.NamespaceEQ(charge.Namespace)).
			SetPaymentTerm(intent.PaymentTerm).
			SetInvoiceAt(intent.InvoiceAt.In(time.UTC)).
			SetDiscounts(discounts).
			SetProRating(proRating).
			SetAmountBeforeProration(intent.AmountBeforeProration).
			SetAmountAfterProration(intent.AmountAfterProration)

		if charge.State.Payment != nil {
			create = create.SetStdInvoicePaymentSettlementID(charge.State.Payment.ID)
		} else {
			create = create.ClearStdInvoicePaymentSettlementID()
		}

		dbFlatFee, err := create.Save(ctx)
		if err != nil {
			return charges.FlatFeeCharge{}, err
		}

		dbEntity.Edges.FlatFee = dbFlatFee
		// We are not expanding the relaizations rather reuse the existing ones
		mapped, err := MapFlatFeeChargeFromDB(dbEntity, charges.ExpandNone)
		if err != nil {
			return charges.FlatFeeCharge{}, err
		}

		// We are just reusing the existing state
		mapped.State = charge.State

		return mapped, nil
	})
}
