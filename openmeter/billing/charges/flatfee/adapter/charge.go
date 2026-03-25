package adapter

import (
	"context"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (a *adapter) UpdateCharge(ctx context.Context, charge flatfee.Charge) error {
	if err := charge.ManagedModel.Validate(); err != nil {
		return err
	}

	if err := charge.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		intent := charge.Intent

		var discounts *productcatalog.Discounts
		if intent.PercentageDiscounts != nil {
			discounts = &productcatalog.Discounts{Percentage: intent.PercentageDiscounts}
		}

		proRating, err := proRatingConfigToDB(intent.ProRating)
		if err != nil {
			return err
		}

		update := tx.db.ChargeFlatFee.UpdateOneID(charge.ID).
			Where(dbchargeflatfee.NamespaceEQ(charge.Namespace)).
			SetPaymentTerm(intent.PaymentTerm).
			SetInvoiceAt(intent.InvoiceAt.In(time.UTC)).
			SetDiscounts(discounts).
			SetProRating(proRating).
			SetAmountBeforeProration(intent.AmountBeforeProration).
			SetAmountAfterProration(intent.AmountAfterProration)

		update, err = chargemeta.Update(update, chargemeta.UpdateInput{
			ManagedResource: charge.ManagedResource,
			Intent:          charge.Intent.Intent,
			Status:          charge.Status,
		})
		if err != nil {
			return err
		}

		_, err = update.Save(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

func (a *adapter) CreateCharges(ctx context.Context, in flatfee.CreateChargesInput) ([]flatfee.Charge, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]flatfee.Charge, error) {
		creates, err := slicesx.MapWithErr(in.Intents, func(intent flatfee.IntentWithInitialStatus) (*db.ChargeFlatFeeCreate, error) {
			return tx.buildCreateFlatFeeCharge(in.Namespace, intent)
		})
		if err != nil {
			return nil, err
		}

		entities, err := tx.db.ChargeFlatFee.CreateBulk(creates...).Save(ctx)
		if err != nil {
			return nil, err
		}

		// Let's reserve the charge IDs
		err = tx.metaAdapter.RegisterCharges(ctx, meta.RegisterChargesInput{
			Namespace: in.Namespace,
			Type:      meta.ChargeTypeFlatFee,
			Charges: lo.Map(entities, func(entity *db.ChargeFlatFee, idx int) meta.IDWithUniqueReferenceID {
				return meta.IDWithUniqueReferenceID{
					ID:                entity.ID,
					UniqueReferenceID: entity.UniqueReferenceID,
				}
			}),
		})
		if err != nil {
			return nil, err
		}

		out := make([]flatfee.Charge, 0, len(entities))
		for _, entity := range entities {
			charge, err := MapChargeFlatFeeFromDB(entity, meta.ExpandNone)
			if err != nil {
				return nil, err
			}
			out = append(out, charge)
		}
		return out, nil
	})
}

func (a *adapter) GetByIDs(ctx context.Context, input flatfee.GetByIDsInput) ([]flatfee.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]flatfee.Charge, error) {
		query := tx.db.ChargeFlatFee.Query().
			Where(dbchargeflatfee.IDIn(
				lo.Map(input.IDs, func(id meta.ChargeID, idx int) string {
					return id.ID
				})...,
			))

		if input.Expands.Has(meta.ExpandRealizations) {
			query = query.WithCreditAllocations().
				WithInvoicedUsage().
				WithPayment()
		}

		entities, err := query.All(ctx)
		if err != nil {
			return nil, err
		}

		entitiesInOrder, err := entutils.InIDOrder(input.IDs.ToNamespacedIDs(), entities)
		if err != nil {
			return nil, err
		}

		return slicesx.MapWithErr(entitiesInOrder, func(entity *db.ChargeFlatFee) (flatfee.Charge, error) {
			return MapChargeFlatFeeFromDB(entity, input.Expands)
		})
	})
}

func (a *adapter) buildCreateFlatFeeCharge(ns string, intent flatfee.IntentWithInitialStatus) (*db.ChargeFlatFeeCreate, error) {
	var discounts *productcatalog.Discounts
	if intent.PercentageDiscounts != nil {
		discounts = &productcatalog.Discounts{Percentage: intent.PercentageDiscounts}
	}

	proRating, err := proRatingConfigToDB(intent.ProRating)
	if err != nil {
		return nil, err
	}

	create := a.db.ChargeFlatFee.Create().
		SetNamespace(ns).
		SetPaymentTerm(intent.PaymentTerm).
		SetInvoiceAt(intent.InvoiceAt.In(time.UTC)).
		SetSettlementMode(intent.SettlementMode).
		SetNillableFeatureKey(lo.EmptyableToPtr(intent.FeatureKey)).
		SetProRating(proRating).
		SetAmountBeforeProration(intent.AmountBeforeProration).
		SetAmountAfterProration(intent.AmountAfterProration)

	if discounts != nil {
		create = create.SetDiscounts(discounts)
	}

	create, err = chargemeta.Create[*db.ChargeFlatFeeCreate](create, chargemeta.CreateInput{
		Namespace: ns,
		Intent:    intent.Intent.Intent,
		Status:    intent.InitialStatus,
	})
	if err != nil {
		return nil, err
	}

	return create, nil
}
