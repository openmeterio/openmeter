package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeusagebased "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebased"
	dbchargeusagebasedruns "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedruns"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ usagebased.ChargeAdapter = (*adapter)(nil)

func (a *adapter) UpdateCharge(ctx context.Context, charge usagebased.ChargeBase) (usagebased.ChargeBase, error) {
	if err := charge.Validate(); err != nil {
		return usagebased.ChargeBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.ChargeBase, error) {
		metaStatus, err := charge.Status.ToMetaChargeStatus()
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		baseIntent := charge.Intent.GetBaseIntent()

		update := tx.db.ChargeUsageBased.UpdateOneID(charge.ID).
			Where(dbchargeusagebased.NamespaceEQ(charge.Namespace)).
			SetDiscounts(&baseIntent.Discounts).
			SetFeatureKey(baseIntent.FeatureKey).
			SetFeatureID(charge.State.FeatureID).
			SetOrClearIntentDeletedAt(convert.TimePtrIn(baseIntent.IntentDeletedAt, time.UTC)).
			SetInvoiceAt(meta.NormalizeTimestamp(baseIntent.InvoiceAt).In(time.UTC)).
			SetPrice(&baseIntent.Price).
			SetRatingEngine(charge.State.RatingEngine).
			SetStatus(metaStatus).
			SetStatusDetailed(charge.Status).
			SetOrClearCurrentRealizationRunID(charge.State.CurrentRealizationRunID)

		update, err = chargemeta.Update(update, chargemeta.UpdateInput{
			ManagedResource:     charge.ManagedResource,
			IntentMutableFields: baseIntent.IntentMutableFields.IntentMutableFields,
			Annotations:         baseIntent.Annotations,
			Status:              metaStatus,
			AdvanceAfter:        meta.NormalizeOptionalTimestamp(charge.State.AdvanceAfter),
		})
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		update = update.SetOrClearDeletedAt(convert.TimePtrIn(charge.GetIntentDeletedAt(), time.UTC))

		dbUpdatedChargeBase, err := update.Save(ctx)
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		if overrideLayer := charge.Intent.GetOverrideLayerMutableFields(); overrideLayer != nil {
			intentOverride, err := tx.updateIntentOverride(ctx, charge.GetChargeID(), overrideLayer)
			if err != nil {
				return usagebased.ChargeBase{}, fmt.Errorf("updating usage based charge override: %w", err)
			}

			dbUpdatedChargeBase.Edges.IntentOverride = intentOverride
		}

		return MapChargeBaseFromDB(dbUpdatedChargeBase), nil
	})
}

func (a *adapter) UpdateSubscriptionItemID(ctx context.Context, charge usagebased.Charge, newSubscriptionItemID string) (usagebased.Charge, error) {
	if err := charge.ManagedModel.Validate(); err != nil {
		return usagebased.Charge{}, err
	}

	if err := charge.Validate(); err != nil {
		return usagebased.Charge{}, err
	}

	if newSubscriptionItemID == "" {
		return usagebased.Charge{}, fmt.Errorf("subscription item ID is required")
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.Charge, error) {
		// TODO: make subscription_item_id immutable again once subscription edits
		// no longer recreate the item ID for logical item updates.
		updatedChargeBase, err := tx.db.ChargeUsageBased.UpdateOneID(charge.ID).
			Where(dbchargeusagebased.NamespaceEQ(charge.Namespace)).
			SetSubscriptionItemID(newSubscriptionItemID).
			Save(ctx)
		if err != nil {
			return usagebased.Charge{}, err
		}

		overrideLayer := charge.Intent.GetOverrideLayerMutableFields()
		charge.ChargeBase = MapChargeBaseFromDB(updatedChargeBase)
		charge.Intent = usagebased.NewOverridableIntent(charge.Intent.GetBaseIntent(), overrideLayer)

		return charge, nil
	})
}

func (a *adapter) DeleteCharge(ctx context.Context, charge usagebased.Charge) error {
	if err := charge.ManagedModel.Validate(); err != nil {
		return err
	}

	if err := charge.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if err := charge.Intent.MutateEffective(func(intentMutableFields *usagebased.IntentMutableFields) {
			intentMutableFields.IntentDeletedAt = lo.ToPtr(clock.Now())
		}); err != nil {
			return err
		}

		charge.DeletedAt = charge.GetIntentDeletedAt()
		charge.Status = usagebased.StatusDeleted

		metaStatus, err := charge.Status.ToMetaChargeStatus()
		if err != nil {
			return err
		}

		baseIntent := charge.Intent.GetBaseIntent()

		update := tx.db.ChargeUsageBased.UpdateOneID(charge.ID).
			Where(dbchargeusagebased.NamespaceEQ(charge.Namespace)).
			SetStatus(metaStatus).
			SetStatusDetailed(charge.Status)

		update, err = chargemeta.Update(update, chargemeta.UpdateInput{
			ManagedResource:     charge.ManagedResource,
			IntentMutableFields: baseIntent.IntentMutableFields.IntentMutableFields,
			Annotations:         baseIntent.Annotations,
			Status:              metaStatus,
			AdvanceAfter:        charge.State.AdvanceAfter,
		})
		if err != nil {
			return err
		}

		update = update.
			SetOrClearIntentDeletedAt(convert.TimePtrIn(baseIntent.IntentDeletedAt, time.UTC)).
			SetOrClearDeletedAt(convert.TimePtrIn(charge.GetIntentDeletedAt(), time.UTC))

		if _, err := update.Save(ctx); err != nil {
			return err
		}

		if overrideLayer := charge.Intent.GetOverrideLayerMutableFields(); overrideLayer != nil {
			if _, err := tx.updateIntentOverride(ctx, charge.GetChargeID(), overrideLayer); err != nil {
				return fmt.Errorf("updating usage based intent override: %w", err)
			}
		}

		return tx.metaAdapter.DeleteRegisteredCharge(ctx, charge.GetChargeID())
	})
}

func (a *adapter) CreateCharges(ctx context.Context, in usagebased.CreateChargesInput) ([]usagebased.Charge, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]usagebased.Charge, error) {
		creates, err := slicesx.MapWithErr(in.Intents, func(intent usagebased.CreateIntent) (*db.ChargeUsageBasedCreate, error) {
			return tx.buildCreateUsageBasedCharge(ctx, in.Namespace, intent)
		})
		if err != nil {
			return nil, err
		}

		entities, err := tx.db.ChargeUsageBased.CreateBulk(creates...).Save(ctx)
		if err != nil {
			return nil, metaadapter.MapChargeConstraintError(err)
		}

		return slicesx.MapWithErr(entities, func(entity *db.ChargeUsageBased) (usagebased.Charge, error) {
			return MapChargeFromDB(entity, meta.ExpandNone)
		})
	})
}

func (a *adapter) GetByIDs(ctx context.Context, input usagebased.GetByIDsInput) ([]usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]usagebased.Charge, error) {
		query := tx.db.ChargeUsageBased.Query().
			// Note: we are skipping the namespace filter here to allow multi-namespace expansions as needed, but InIDOrder filters for namespaces.
			Where(dbchargeusagebased.Namespace(input.Namespace)).
			Where(dbchargeusagebased.IDIn(input.IDs...)).
			WithIntentOverride()

		if input.Expands.Has(meta.ExpandRealizations) {
			query = expandRealizations(query, input.Expands)
		}

		entities, err := query.All(ctx)
		if err != nil {
			return nil, err
		}

		entitiesInOrder, err := entutils.InIDOrder(input.Namespace, input.IDs, entities)
		if err != nil {
			return nil, err
		}

		out, err := slicesx.MapWithErr(entitiesInOrder, func(entity *db.ChargeUsageBased) (usagebased.Charge, error) {
			return MapChargeFromDB(entity, input.Expands)
		})
		if err != nil {
			return nil, err
		}

		if input.Expands.Has(meta.ExpandDetailedLines) {
			out, err = slicesx.MapWithErr(out, func(charge usagebased.Charge) (usagebased.Charge, error) {
				return tx.FetchDetailedLines(ctx, charge)
			})
			if err != nil {
				return nil, err
			}
		}

		return out, nil
	})
}

func (a *adapter) GetByID(ctx context.Context, input usagebased.GetByIDInput) (usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return usagebased.Charge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.Charge, error) {
		query := tx.db.ChargeUsageBased.Query().
			Where(dbchargeusagebased.Namespace(input.ChargeID.Namespace)).
			Where(dbchargeusagebased.ID(input.ChargeID.ID)).
			WithIntentOverride()

		if input.Expands.Has(meta.ExpandRealizations) {
			query = expandRealizations(query, input.Expands)
		}

		entity, err := query.First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return usagebased.Charge{}, models.NewGenericNotFoundError(fmt.Errorf("usage based charge [id=%s] not found", input.ChargeID))
			}

			return usagebased.Charge{}, fmt.Errorf("querying usage based charge [id=%s]: %w", input.ChargeID, err)
		}

		charge, err := MapChargeFromDB(entity, input.Expands)
		if err != nil {
			return usagebased.Charge{}, err
		}

		if input.Expands.Has(meta.ExpandDetailedLines) {
			charge, err = tx.FetchDetailedLines(ctx, charge)
			if err != nil {
				return usagebased.Charge{}, err
			}
		}

		return charge, nil
	})
}

func expandRealizations(query *db.ChargeUsageBasedQuery, expands meta.Expands) *db.ChargeUsageBasedQuery {
	return query.WithRuns(
		func(runs *db.ChargeUsageBasedRunsQuery) {
			if !expands.Has(meta.ExpandDeletedRealizations) {
				runs = runs.Where(dbchargeusagebasedruns.DeletedAtIsNil())
			}

			runs.WithCreditAllocations().
				WithInvoicedUsage().
				WithPayment()
		},
	)
}

func (a *adapter) buildCreateUsageBasedCharge(ctx context.Context, ns string, intent usagebased.CreateIntent) (*db.ChargeUsageBasedCreate, error) {
	baseIntent := intent.Intent.GetBaseIntent()

	create := a.db.ChargeUsageBased.Create().
		SetNillableDeletedAt(convert.TimePtrIn(baseIntent.IntentDeletedAt, time.UTC)).
		SetNillableIntentDeletedAt(convert.TimePtrIn(baseIntent.IntentDeletedAt, time.UTC)).
		SetDiscounts(&baseIntent.Discounts).
		SetFeatureID(intent.FeatureID).
		SetRatingEngine(intent.RatingEngine).
		SetPrice(&baseIntent.Price).
		SetStatusDetailed(usagebased.Status(meta.ChargeStatusCreated)).
		SetFeatureKey(baseIntent.FeatureKey).
		SetInvoiceAt(meta.NormalizeTimestamp(baseIntent.InvoiceAt).In(time.UTC)).
		SetSettlementMode(baseIntent.SettlementMode)

	create, err := chargemeta.Create[*db.ChargeUsageBasedCreate](create, chargemeta.CreateInput{
		Namespace:           ns,
		Intent:              baseIntent.Intent,
		IntentMutableFields: baseIntent.IntentMutableFields.IntentMutableFields,
		Status:              meta.ChargeStatusCreated,
	})
	if err != nil {
		return nil, err
	}

	return create, nil
}
