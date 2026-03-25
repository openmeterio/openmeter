package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeusagebased "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebased"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ usagebased.ChargeAdapter = (*adapter)(nil)

func (a *adapter) UpdateStatus(ctx context.Context, input usagebased.UpdateStatusInput) (usagebased.ChargeBase, error) {
	if err := input.Validate(); err != nil {
		return usagebased.ChargeBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.ChargeBase, error) {
		metaStatus, err := input.Status.ToMetaChargeStatus()
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		dbUpdatedChargeBase, err := tx.db.ChargeUsageBased.UpdateOneID(input.Charge.ID).
			Where(dbchargeusagebased.NamespaceEQ(input.Charge.Namespace)).
			SetStatus(metaStatus).
			SetStatusDetailed(input.Status).
			Save(ctx)
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		return MapChargeBaseFromDB(dbUpdatedChargeBase), nil
	})
}

func (a *adapter) UpdateCharge(ctx context.Context, charge usagebased.ChargeBase) (usagebased.ChargeBase, error) {
	if err := charge.Validate(); err != nil {
		return usagebased.ChargeBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.ChargeBase, error) {
		metaStatus, err := charge.Status.ToMetaChargeStatus()
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		update := tx.db.ChargeUsageBased.UpdateOneID(charge.ID).
			Where(dbchargeusagebased.NamespaceEQ(charge.Namespace)).
			SetDiscounts(&charge.Intent.Discounts).
			SetStatus(metaStatus).
			SetStatusDetailed(charge.Status).
			SetOrClearCurrentRealizationRunID(charge.State.CurrentRealizationRunID)

		update, err = chargemeta.Update(update, chargemeta.UpdateInput{
			ManagedResource: charge.ManagedResource,
			Intent:          charge.Intent.Intent,
			Status:          metaStatus,
			AdvanceAfter:    charge.State.AdvanceAfter,
		})
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		dbUpdatedChargeBase, err := update.Save(ctx)
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		return MapChargeBaseFromDB(dbUpdatedChargeBase), nil
	})
}

func (a *adapter) CreateCharges(ctx context.Context, in usagebased.CreateInput) ([]usagebased.Charge, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]usagebased.Charge, error) {
		creates, err := slicesx.MapWithErr(in.Intents, func(intent usagebased.Intent) (*db.ChargeUsageBasedCreate, error) {
			return tx.buildCreateUsageBasedCharge(ctx, in.Namespace, intent)
		})
		if err != nil {
			return nil, err
		}

		entities, err := tx.db.ChargeUsageBased.CreateBulk(creates...).Save(ctx)
		if err != nil {
			return nil, err
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
			Where(dbchargeusagebased.IDIn(input.IDs...))

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

		return slicesx.MapWithErr(entitiesInOrder, func(entity *db.ChargeUsageBased) (usagebased.Charge, error) {
			return MapChargeFromDB(entity, input.Expands)
		})
	})
}

func (a *adapter) GetByID(ctx context.Context, input usagebased.GetByIDInput) (usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return usagebased.Charge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.Charge, error) {
		query := tx.db.ChargeUsageBased.Query().
			Where(dbchargeusagebased.Namespace(input.ChargeID.Namespace)).
			Where(dbchargeusagebased.ID(input.ChargeID.ID))

		if input.Expands.Has(meta.ExpandRealizations) {
			query = expandRealizations(query)
		}

		entity, err := query.First(ctx)
		if err != nil {
			return usagebased.Charge{}, fmt.Errorf("querying usage based charge [id=%s]: %w", input.ChargeID, err)
		}

		return MapChargeFromDB(entity, input.Expands)
	})
}

func expandRealizations(query *db.ChargeUsageBasedQuery) *db.ChargeUsageBasedQuery {
	return query.WithRuns(
		func(runs *db.ChargeUsageBasedRunsQuery) {
			runs.WithCreditAllocations().
				WithInvoicedUsage().
				WithPayment()
		},
	)
}

func (a *adapter) buildCreateUsageBasedCharge(ctx context.Context, ns string, intent usagebased.Intent) (*db.ChargeUsageBasedCreate, error) {
	create := a.db.ChargeUsageBased.Create().
		SetDiscounts(&intent.Discounts).
		SetPrice(&intent.Price).
		SetStatusDetailed(usagebased.Status(meta.ChargeStatusCreated)).
		SetFeatureKey(intent.FeatureKey).
		SetInvoiceAt(intent.InvoiceAt.In(time.UTC)).
		SetSettlementMode(intent.SettlementMode)

	create, err := chargemeta.Create[*db.ChargeUsageBasedCreate](create, chargemeta.CreateInput{
		Namespace: ns,
		Intent:    intent.Intent,
		Status:    meta.ChargeStatusCreated,
	})
	if err != nil {
		return nil, err
	}

	return create, nil
}
