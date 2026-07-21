package adapter

import (
	"context"
	"fmt"

	"github.com/lib/pq"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargecreditpurchase "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ creditpurchase.Adapter = (*adapter)(nil)

func (a *adapter) UpdateCharge(ctx context.Context, charge creditpurchase.ChargeBase) (creditpurchase.ChargeBase, error) {
	if err := charge.Validate(); err != nil {
		return creditpurchase.ChargeBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.ChargeBase, error) {
		metaStatus, err := charge.Status.ToMetaChargeStatus()
		if err != nil {
			return creditpurchase.ChargeBase{}, err
		}

		update := tx.db.ChargeCreditPurchase.UpdateOneID(charge.ID).
			Where(dbchargecreditpurchase.NamespaceEQ(charge.Namespace)).
			SetCreditAmount(charge.Intent.CreditAmount).
			SetSettlement(charge.Intent.Settlement).
			SetStatusDetailed(charge.Status)

		update, err = chargemeta.Update(update, chargemeta.UpdateInput{
			ManagedResource:     charge.ManagedResource,
			Intent:              charge.Intent.Intent,
			IntentMutableFields: charge.Intent.IntentMutableFields.IntentMutableFields,
			Status:              metaStatus,
		})
		if err != nil {
			return creditpurchase.ChargeBase{}, err
		}

		dbCreditPurchase, err := update.Save(ctx)
		if err != nil {
			return creditpurchase.ChargeBase{}, err
		}

		return fromDBBaseWithCurrency(dbCreditPurchase, charge.Intent.Currency)
	})
}

func (a *adapter) CreateCharge(ctx context.Context, in creditpurchase.CreateInput) (creditpurchase.Charge, error) {
	if err := in.Validate(); err != nil {
		return creditpurchase.Charge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.Charge, error) {
		initialStatus := creditpurchase.StatusCreated

		metaStatus, err := initialStatus.ToMetaChargeStatus()
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		create := tx.db.ChargeCreditPurchase.Create().
			SetNamespace(in.Namespace).
			SetCreditAmount(in.Intent.CreditAmount).
			SetNillableEffectiveAt(meta.NormalizeOptionalTimestamp(in.Intent.EffectiveAt)).
			SetNillableExpiresAt(meta.NormalizeOptionalTimestamp(in.Intent.ExpiresAt)).
			SetNillablePriority(in.Intent.Priority).
			SetFeatureFilters(pq.StringArray(in.Intent.FeatureFilters.Normalize())).
			SetSettlement(in.Intent.Settlement).
			SetNillableKey(in.Intent.Key).
			SetStatusDetailed(initialStatus)

		create, err = chargemeta.Create(create, chargemeta.CreateInput{
			Namespace:           in.Namespace,
			Intent:              in.Intent.Intent,
			IntentMutableFields: in.Intent.IntentMutableFields.IntentMutableFields,
			Status:              metaStatus,
		})
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		dbCreditPurchase, err := create.Save(ctx)
		if err != nil {
			return creditpurchase.Charge{}, metaadapter.MapChargeConstraintError(err)
		}

		err = tx.metaAdapter.RegisterCharges(ctx, meta.RegisterChargesInput{
			Namespace: in.Namespace,
			Type:      meta.ChargeTypeCreditPurchase,
			Charges: []meta.IDWithUniqueReferenceID{
				{
					ID:                dbCreditPurchase.ID,
					UniqueReferenceID: dbCreditPurchase.UniqueReferenceID,
				},
			},
		})
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		return FromDBWithCurrency(dbCreditPurchase, in.Intent.Currency, meta.ExpandNone)
	})
}

func (a *adapter) MarkVoided(ctx context.Context, input creditpurchase.MarkVoidedAdapterInput) (creditpurchase.ChargeBase, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.ChargeBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.ChargeBase, error) {
		dbCreditPurchase, err := tx.db.ChargeCreditPurchase.UpdateOneID(input.Charge.ID).
			Where(dbchargecreditpurchase.NamespaceEQ(input.Charge.Namespace)).
			SetVoidedAt(input.VoidedAt).
			Save(ctx)
		if err != nil {
			return creditpurchase.ChargeBase{}, fmt.Errorf("marking credit purchase charge voided [id=%s]: %w", input.Charge.ID, err)
		}

		return fromDBBaseWithCurrency(dbCreditPurchase, input.Charge.Intent.Currency)
	})
}

func (a *adapter) GetByID(ctx context.Context, input creditpurchase.GetByIDInput) (creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.Charge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.Charge, error) {
		query := tx.db.ChargeCreditPurchase.Query().
			Where(
				dbchargecreditpurchase.Namespace(input.ChargeID.Namespace),
				dbchargecreditpurchase.ID(input.ChargeID.ID),
			)

		query = withExpands(query, input.Expands)

		entity, err := query.Only(ctx)
		if err != nil {
			return creditpurchase.Charge{}, fmt.Errorf("getting credit purchase charge [id=%s]: %w", input.ChargeID.ID, err)
		}

		return FromDB(entity, input.Expands)
	})
}

func (a *adapter) GetByIDs(ctx context.Context, input creditpurchase.GetByIDsInput) ([]creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]creditpurchase.Charge, error) {
		query := tx.db.ChargeCreditPurchase.Query().
			Where(dbchargecreditpurchase.Namespace(input.Namespace)).
			Where(dbchargecreditpurchase.IDIn(input.IDs...))

		query = withExpands(query, input.Expands)

		entities, err := query.All(ctx)
		if err != nil {
			return nil, err
		}

		entitiesInOrder, err := entutils.InIDOrder(input.Namespace, input.IDs, entities)
		if err != nil {
			return nil, err
		}

		return slicesx.MapWithErr(entitiesInOrder, func(entity *db.ChargeCreditPurchase) (creditpurchase.Charge, error) {
			return FromDB(entity, input.Expands)
		})
	})
}

func (a *adapter) ListCharges(ctx context.Context, input creditpurchase.ListChargesInput) (pagination.Result[creditpurchase.Charge], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[creditpurchase.Charge]{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (pagination.Result[creditpurchase.Charge], error) {
		query := tx.db.ChargeCreditPurchase.Query().
			Where(dbchargecreditpurchase.Namespace(input.Namespace))

		if !input.IncludeDeleted {
			query = query.Where(dbchargecreditpurchase.DeletedAtIsNil())
		}

		if len(input.CustomerIDs) > 0 {
			query = query.Where(dbchargecreditpurchase.CustomerIDIn(input.CustomerIDs...))
		}

		if len(input.Statuses) > 0 {
			query = query.Where(dbchargecreditpurchase.StatusIn(input.Statuses...))
		}

		if len(input.Currencies) > 0 {
			query = query.Where(
				dbchargecreditpurchase.Or(
					dbchargecreditpurchase.FiatCurrencyCodeIn(input.Currencies...),
					hasCustomCurrencyCode(input.Namespace, input.Currencies...),
				),
			)
		}

		if input.Voided != nil {
			if *input.Voided {
				query = query.Where(dbchargecreditpurchase.VoidedAtNotNil())
			} else {
				query = query.Where(dbchargecreditpurchase.VoidedAtIsNil())
			}
		}

		if input.Expiration != nil {
			if input.Expiration.Expired {
				query = query.Where(dbchargecreditpurchase.ExpiresAtLTE(input.Expiration.AsOf))
			} else {
				query = query.Where(dbchargecreditpurchase.Or(
					dbchargecreditpurchase.ExpiresAtIsNil(),
					dbchargecreditpurchase.ExpiresAtGT(input.Expiration.AsOf),
				))
			}
		}

		query = filter.ApplyToQuery(query, input.Key, dbchargecreditpurchase.FieldKey)

		query = withExpands(query, input.Expands)

		res, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return pagination.Result[creditpurchase.Charge]{}, err
		}

		charges, err := slicesx.MapWithErr(res.Items, func(entity *db.ChargeCreditPurchase) (creditpurchase.Charge, error) {
			return FromDB(entity, input.Expands)
		})
		if err != nil {
			return pagination.Result[creditpurchase.Charge]{}, err
		}

		return pagination.Result[creditpurchase.Charge]{
			Page:       res.Page,
			TotalCount: res.TotalCount,
			Items:      charges,
		}, nil
	})
}

func withExpands(query *db.ChargeCreditPurchaseQuery, expands meta.Expands) *db.ChargeCreditPurchaseQuery {
	query = query.WithCustomCurrency()

	if expands.Has(meta.ExpandRealizations) {
		query = query.WithCreditGrant().WithExternalPayment().WithInvoicedPayment()
	}
	return query
}
