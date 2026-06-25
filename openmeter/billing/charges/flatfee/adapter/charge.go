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
	dbchargeflatfeeoverride "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeeoverride"
	dbchargeflatfeerun "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeerun"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
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

		intent := charge.Intent.GetBaseIntent()

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
			SetOrClearIntentDeletedAt(convert.TimePtrIn(intent.IntentDeletedAt, time.UTC)).
			SetInvoiceAt(meta.NormalizeTimestamp(intent.InvoiceAt).In(time.UTC)).
			SetDiscounts(discounts).
			SetOrClearFeatureID(charge.State.FeatureID).
			SetProRating(proRating).
			SetStatusDetailed(charge.Status).
			SetAmountBeforeProration(intent.AmountBeforeProration).
			SetAmountAfterProration(charge.State.AmountAfterProration)

		update, err = chargemeta.Update(update, chargemeta.UpdateInput{
			ManagedResource:     charge.ManagedResource,
			IntentMutableFields: intent.IntentMutableFields.IntentMutableFields,
			Annotations:         intent.Annotations,
			Status:              metaStatus,
			AdvanceAfter:        meta.NormalizeOptionalTimestamp(charge.State.AdvanceAfter),
		})
		if err != nil {
			return flatfee.ChargeBase{}, err
		}

		update = update.SetOrClearDeletedAt(convert.TimePtrIn(charge.Intent.GetDeletedAt(), time.UTC))

		dbUpdatedChargeBase, err := update.Save(ctx)
		if err != nil {
			return flatfee.ChargeBase{}, err
		}

		if overrideLayer := charge.Intent.GetOverrideLayerMutableFields(); overrideLayer != nil {
			intentOverride, err := tx.updateIntentOverride(ctx, charge.GetChargeID(), overrideLayer)
			if err != nil {
				return flatfee.ChargeBase{}, fmt.Errorf("updating flat fee charge override: %w", err)
			}

			dbUpdatedChargeBase.Edges.IntentOverride = intentOverride
		}

		return MapChargeBaseFromDB(dbUpdatedChargeBase), nil
	})
}

func (a *adapter) UpdateSubscriptionItemID(ctx context.Context, charge flatfee.Charge, newSubscriptionItemID string) (flatfee.Charge, error) {
	if err := charge.ManagedModel.Validate(); err != nil {
		return flatfee.Charge{}, err
	}

	if err := charge.Validate(); err != nil {
		return flatfee.Charge{}, err
	}

	if newSubscriptionItemID == "" {
		return flatfee.Charge{}, fmt.Errorf("subscription item ID is required")
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.Charge, error) {
		// TODO: make subscription_item_id immutable again once subscription edits
		// no longer recreate the item ID for logical item updates.
		updatedChargeBase, err := tx.db.ChargeFlatFee.UpdateOneID(charge.ID).
			Where(dbchargeflatfee.NamespaceEQ(charge.Namespace)).
			SetSubscriptionItemID(newSubscriptionItemID).
			Save(ctx)
		if err != nil {
			return flatfee.Charge{}, err
		}

		override, err := tx.db.ChargeFlatFeeOverride.Query().
			Where(dbchargeflatfeeoverride.NamespaceEQ(charge.Namespace)).
			Where(dbchargeflatfeeoverride.ChargeIDEQ(charge.ID)).
			First(ctx)
		if err != nil && !db.IsNotFound(err) {
			return flatfee.Charge{}, err
		}

		updatedChargeBase.Edges.IntentOverride = override
		charge.ChargeBase = MapChargeBaseFromDB(updatedChargeBase)

		return charge, nil
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

		err := charge.Intent.MutateEffective(func(intentMutableFields flatfee.IntentMutableFields) (flatfee.IntentMutableFields, error) {
			intentMutableFields.IntentDeletedAt = lo.ToPtr(clock.Now())
			return intentMutableFields, nil
		})
		if err != nil {
			return err
		}

		charge.DeletedAt = charge.Intent.GetDeletedAt()
		charge.Status = flatfee.StatusDeleted

		metaStatus, err := charge.Status.ToMetaChargeStatus()
		if err != nil {
			return err
		}

		update = update.SetStatusDetailed(charge.Status)

		baseIntent := charge.Intent.GetBaseIntent()

		update, err = chargemeta.Update(update, chargemeta.UpdateInput{
			ManagedResource:     charge.ManagedResource,
			IntentMutableFields: baseIntent.IntentMutableFields.IntentMutableFields,
			Annotations:         baseIntent.Annotations,
			Status:              metaStatus,
		})
		if err != nil {
			return err
		}

		update = update.
			SetOrClearIntentDeletedAt(convert.TimePtrIn(baseIntent.IntentDeletedAt, time.UTC)).
			SetOrClearDeletedAt(convert.TimePtrIn(charge.Intent.GetDeletedAt(), time.UTC))

		if _, err := update.Save(ctx); err != nil {
			return err
		}

		if overrideLayer := charge.Intent.GetOverrideLayerMutableFields(); overrideLayer != nil {
			if _, err := tx.updateIntentOverride(ctx, charge.GetChargeID(), overrideLayer); err != nil {
				return fmt.Errorf("updating flat fee intent override: %w", err)
			}
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
			Where(dbchargeflatfee.IDIn(input.IDs...)).
			WithIntentOverride()

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
				return tx.FetchCurrentRunDetailedLines(ctx, charge)
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
			Where(dbchargeflatfee.ID(input.ChargeID.ID)).
			WithIntentOverride()

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
			return tx.FetchCurrentRunDetailedLines(ctx, charge)
		}

		return charge, nil
	})
}

func expandRealizations(query *db.ChargeFlatFeeQuery) *db.ChargeFlatFeeQuery {
	return query.WithRuns(func(query *db.ChargeFlatFeeRunQuery) {
		query.
			Order(
				dbchargeflatfeerun.ByServicePeriodTo(),
				dbchargeflatfeerun.ByCreatedAt(),
			).
			WithCreditAllocations().
			WithInvoicedUsage().
			WithPayment()
	})
}

func (a *adapter) buildCreateFlatFeeCharge(ns string, intentWithStatus flatfee.IntentWithInitialStatus) (*db.ChargeFlatFeeCreate, error) {
	metaStatus, err := intentWithStatus.InitialStatus.ToMetaChargeStatus()
	if err != nil {
		return nil, err
	}

	intent := intentWithStatus.Intent

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
		SetNillableDeletedAt(convert.TimePtrIn(intent.IntentDeletedAt, time.UTC)).
		SetNillableIntentDeletedAt(convert.TimePtrIn(intent.IntentDeletedAt, time.UTC)).
		SetPaymentTerm(intent.PaymentTerm).
		SetInvoiceAt(meta.NormalizeTimestamp(intent.InvoiceAt).In(time.UTC)).
		SetSettlementMode(intent.SettlementMode).
		SetNillableFeatureID(intentWithStatus.FeatureID).
		SetNillableFeatureKey(lo.EmptyableToPtr(intent.FeatureKey)).
		SetStatusDetailed(intentWithStatus.InitialStatus).
		SetProRating(proRating).
		SetAmountBeforeProration(intent.AmountBeforeProration).
		SetAmountAfterProration(intentWithStatus.AmountAfterProration)

	if discounts != nil {
		create = create.SetDiscounts(discounts)
	}

	create, err = chargemeta.Create[*db.ChargeFlatFeeCreate](create, chargemeta.CreateInput{
		Namespace:           ns,
		Intent:              intent.Intent,
		IntentMutableFields: intent.IntentMutableFields.IntentMutableFields,
		Status:              metaStatus,
		AdvanceAfter:        meta.NormalizeOptionalTimestamp(intentWithStatus.InitialAdvanceAfter),
	})
	if err != nil {
		return nil, err
	}

	return create, nil
}
