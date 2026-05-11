package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ flatfee.ChargeAdapter = (*adapter)(nil)

func (a *adapter) UpdateCharge(ctx context.Context, charge flatfee.ChargeBase) (flatfee.ChargeBase, error) {
	if err := charge.ManagedModel.Validate(); err != nil {
		return flatfee.ChargeBase{}, err
	}

	if err := charge.Validate(); err != nil {
		return flatfee.ChargeBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.ChargeBase, error) {
		metaStatus, err := charge.Status.ToMetaChargeStatus()
		if err != nil {
			return flatfee.ChargeBase{}, err
		}

		intent := charge.Intent

		var discounts *productcatalog.Discounts
		if intent.PercentageDiscounts != nil {
			discounts = &productcatalog.Discounts{Percentage: intent.PercentageDiscounts}
		}

		proRating, err := proRatingConfigToDB(intent.ProRating)
		if err != nil {
			return flatfee.ChargeBase{}, err
		}

		update := tx.db.ChargeFlatFee.UpdateOneID(charge.ID).
			Where(dbchargeflatfee.NamespaceEQ(charge.Namespace)).
			SetPaymentTerm(intent.PaymentTerm).
			SetInvoiceAt(meta.NormalizeTimestamp(intent.InvoiceAt).In(time.UTC)).
			SetDiscounts(discounts).
			SetOrClearFeatureID(charge.State.FeatureID).
			SetProRating(proRating).
			SetStatusDetailed(charge.Status).
			SetAmountBeforeProration(intent.AmountBeforeProration).
			SetAmountAfterProration(charge.State.AmountAfterProration)

		update, err = chargemeta.Update(update, chargemeta.UpdateInput{
			ManagedResource: charge.ManagedResource,
			Intent:          intent.Intent,
			Status:          metaStatus,
			AdvanceAfter:    meta.NormalizeOptionalTimestamp(charge.State.AdvanceAfter),
		})
		if err != nil {
			return flatfee.ChargeBase{}, err
		}

		dbUpdatedChargeBase, err := update.Save(ctx)
		if err != nil {
			return flatfee.ChargeBase{}, err
		}

		return MapChargeBaseFromDB(dbUpdatedChargeBase), nil
	})
}

func (a *adapter) DeleteCharge(ctx context.Context, charge flatfee.Charge) error {
	if err := charge.ManagedModel.Validate(); err != nil {
		return err
	}

	if err := charge.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		update := tx.db.ChargeFlatFee.UpdateOneID(charge.ID).
			Where(dbchargeflatfee.NamespaceEQ(charge.Namespace))

		charge.DeletedAt = lo.ToPtr(clock.Now())
		charge.Status = flatfee.StatusDeleted

		metaStatus, err := charge.Status.ToMetaChargeStatus()
		if err != nil {
			return err
		}

		update = update.SetStatusDetailed(charge.Status)

		update, err = chargemeta.Update(update, chargemeta.UpdateInput{
			ManagedResource: charge.ManagedResource,
			Intent:          charge.Intent.Intent,
			Status:          metaStatus,
		})
		if err != nil {
			return err
		}

		if _, err := update.Save(ctx); err != nil {
			return err
		}

		return tx.metaAdapter.DeleteRegisteredCharge(ctx, charge.GetChargeID())
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
			Where(dbchargeflatfee.Namespace(input.Namespace)).
			Where(dbchargeflatfee.IDIn(input.IDs...))

		if input.Expands.Has(meta.ExpandRealizations) {
			query = expandRealizations(query)
		}

		entities, err := query.All(ctx)
		if err != nil {
			return nil, err
		}

		entitiesInOrder, err := entutils.InIDOrder(input.Namespace, input.IDs, entities)
		if err != nil {
			return nil, err
		}

		out, err := slicesx.MapWithErr(entitiesInOrder, func(entity *db.ChargeFlatFee) (flatfee.Charge, error) {
			return MapChargeFlatFeeFromDB(entity, input.Expands)
		})
		if err != nil {
			return nil, err
		}

		if input.Expands.Has(meta.ExpandDetailedLines) {
			return slicesx.MapWithErr(out, func(charge flatfee.Charge) (flatfee.Charge, error) {
				return tx.FetchDetailedLines(ctx, charge)
			})
		}

		return out, nil
	})
}

func (a *adapter) GetByID(ctx context.Context, input flatfee.GetByIDInput) (flatfee.Charge, error) {
	if err := input.Validate(); err != nil {
		return flatfee.Charge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.Charge, error) {
		query := tx.db.ChargeFlatFee.Query().
			Where(dbchargeflatfee.Namespace(input.ChargeID.Namespace)).
			Where(dbchargeflatfee.ID(input.ChargeID.ID))

		if input.Expands.Has(meta.ExpandRealizations) {
			query = expandRealizations(query)
		}

		entity, err := query.First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return flatfee.Charge{}, models.NewGenericNotFoundError(fmt.Errorf("flat fee charge [id=%s] not found", input.ChargeID))
			}

			return flatfee.Charge{}, fmt.Errorf("querying flat fee charge [id=%s]: %w", input.ChargeID, err)
		}

		charge, err := MapChargeFlatFeeFromDB(entity, input.Expands)
		if err != nil {
			return flatfee.Charge{}, err
		}

		if input.Expands.Has(meta.ExpandDetailedLines) {
			return tx.FetchDetailedLines(ctx, charge)
		}

		return charge, nil
	})
}

func expandRealizations(query *db.ChargeFlatFeeQuery) *db.ChargeFlatFeeQuery {
	return query.WithCreditAllocations().
		WithInvoicedUsage().
		WithPayment()
}

func (a *adapter) buildCreateFlatFeeCharge(ns string, intent flatfee.IntentWithInitialStatus) (*db.ChargeFlatFeeCreate, error) {
	metaStatus, err := intent.InitialStatus.ToMetaChargeStatus()
	if err != nil {
		return nil, err
	}

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
		SetInvoiceAt(meta.NormalizeTimestamp(intent.InvoiceAt).In(time.UTC)).
		SetSettlementMode(intent.SettlementMode).
		SetNillableFeatureID(intent.FeatureID).
		SetNillableFeatureKey(lo.EmptyableToPtr(intent.FeatureKey)).
		SetStatusDetailed(intent.InitialStatus).
		SetProRating(proRating).
		SetAmountBeforeProration(intent.AmountBeforeProration).
		SetAmountAfterProration(intent.AmountAfterProration)

	if discounts != nil {
		create = create.SetDiscounts(discounts)
	}

	create, err = chargemeta.Create[*db.ChargeFlatFeeCreate](create, chargemeta.CreateInput{
		Namespace:    ns,
		Intent:       intent.Intent.Intent,
		Status:       metaStatus,
		AdvanceAfter: meta.NormalizeOptionalTimestamp(intent.InitialAdvanceAfter),
	})
	if err != nil {
		return nil, err
	}

	return create, nil
}
