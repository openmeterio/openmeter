package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	dbcharge "github.com/openmeterio/openmeter/openmeter/ent/db/charge"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/samber/lo"
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

		entity, err := tx.db.Charge.UpdateOneID(charge.ID).
			Where(dbcharge.NamespaceEQ(charge.Namespace)).
			SetName(intent.Name).
			SetNillableDescription(intent.Description).
			SetServicePeriodFrom(intent.ServicePeriod.From.UTC()).
			SetServicePeriodTo(intent.ServicePeriod.To.UTC()).
			SetBillingPeriodFrom(intent.BillingPeriod.From.UTC()).
			SetBillingPeriodTo(intent.BillingPeriod.To.UTC()).
			SetFullServicePeriodFrom(intent.FullServicePeriod.From.UTC()).
			SetFullServicePeriodTo(intent.FullServicePeriod.To.UTC()).
			SetStatus(charge.Status).
			SetManagedBy(intent.ManagedBy).
			SetNillableUniqueReferenceID(intent.UniqueReferenceID).
			SetMetadata(intent.Metadata).
			SetAnnotations(intent.Annotations).
			Save(ctx)
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

		var authorizedTransactionGroupID *string
		if charge.State.AuthorizedTransaction != nil {
			authorizedTransactionGroupID = lo.EmptyableToPtr(charge.State.AuthorizedTransaction.TransactionGroupID)
		}

		var settledTransactionGroupID *string
		if charge.State.SettledTransaction != nil {
			settledTransactionGroupID = lo.EmptyableToPtr(charge.State.SettledTransaction.TransactionGroupID)
		}

		flatFee, err := tx.db.ChargeFlatFee.UpdateOneID(charge.ID).
			Where(dbchargeflatfee.NamespaceEQ(charge.Namespace)).
			SetPaymentTerm(intent.PaymentTerm).
			SetInvoiceAt(intent.InvoiceAt).
			SetDiscounts(discounts).
			SetProRating(proRating).
			SetAmountBeforeProration(intent.AmountBeforeProration).
			SetAmountAfterProration(intent.AmountAfterProration).
			SetNillableAuthorizedTransactionGroupID(authorizedTransactionGroupID).
			SetNillableSettledTransactionGroupID(settledTransactionGroupID).
			Save(ctx)
		if err != nil {
			return charges.FlatFeeCharge{}, err
		}

		entity.Edges.FlatFee = flatFee
		mapped, err := MapFlatFeeChargeFromDB(entity)
		if err != nil {
			return charges.FlatFeeCharge{}, err
		}

		// We are not updating the credit realizations here
		mapped.State.CreditRealizations = charge.State.CreditRealizations

		return mapped, nil
	})
}
