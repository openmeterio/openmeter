package adapter

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargecreditpurchase "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ creditpurchase.Adapter = (*adapter)(nil)

func (a *adapter) UpdateCharge(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	if err := charge.Validate(); err != nil {
		return creditpurchase.Charge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.Charge, error) {
		update := tx.db.ChargeCreditPurchase.UpdateOneID(charge.ID).
			Where(dbchargecreditpurchase.NamespaceEQ(charge.Namespace)).
			SetCreditAmount(charge.Intent.CreditAmount).
			SetSettlement(charge.Intent.Settlement)

		update, err := chargemeta.Update(update, chargemeta.UpdateInput{
			ManagedResource: charge.ManagedResource,
			Intent:          charge.Intent.Intent,
			Status:          charge.Status,
		})
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		if charge.State.CreditGrantRealization != nil {
			update = update.
				SetCreditGrantTransactionGroupID(charge.State.CreditGrantRealization.TransactionGroupID).
				SetCreditGrantedAt(charge.State.CreditGrantRealization.Time.In(time.UTC))
		}

		dbCreditPurchase, err := update.Save(ctx)
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		mapped, err := MapCreditPurchaseChargeFromDB(dbCreditPurchase, meta.ExpandNone)
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		mapped.State.ExternalPaymentSettlement = charge.State.ExternalPaymentSettlement
		mapped.State.InvoiceSettlement = charge.State.InvoiceSettlement

		return mapped, nil
	})
}

func (a *adapter) CreateCharge(ctx context.Context, in creditpurchase.CreateChargeInput) (creditpurchase.Charge, error) {
	if err := in.Validate(); err != nil {
		return creditpurchase.Charge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.Charge, error) {
		create := tx.db.ChargeCreditPurchase.Create().
			SetNamespace(in.Namespace).
			SetCreditAmount(in.Intent.CreditAmount).
			SetNillableEffectiveAt(meta.NormalizeOptionalTimestamp(in.Intent.EffectiveAt)).
			SetNillablePriority(in.Intent.Priority).
			SetSettlement(in.Intent.Settlement)

		create, err := chargemeta.Create(create, chargemeta.CreateInput{
			Namespace: in.Namespace,
			Intent:    in.Intent.Intent,
			Status:    meta.ChargeStatusCreated,
		})
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		dbCreditPurchase, err := create.Save(ctx)
		if err != nil {
			return creditpurchase.Charge{}, err
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

		return MapCreditPurchaseChargeFromDB(dbCreditPurchase, meta.ExpandNone)
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

		if input.Expands.Has(meta.ExpandRealizations) {
			query = query.WithExternalPayment().WithInvoicedPayment()
		}

		entities, err := query.All(ctx)
		if err != nil {
			return nil, err
		}

		entitiesInOrder, err := entutils.InIDOrder(input.Namespace, input.IDs, entities)
		if err != nil {
			return nil, err
		}

		return slicesx.MapWithErr(entitiesInOrder, func(entity *db.ChargeCreditPurchase) (creditpurchase.Charge, error) {
			return MapCreditPurchaseChargeFromDB(entity, input.Expands)
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
			query = query.Where(dbchargecreditpurchase.CurrencyIn(input.Currencies...))
		}

		if input.Expands.Has(meta.ExpandRealizations) {
			query = query.WithExternalPayment().WithInvoicedPayment()
		}

		res, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return pagination.Result[creditpurchase.Charge]{}, err
		}

		charges, err := slicesx.MapWithErr(res.Items, func(entity *db.ChargeCreditPurchase) (creditpurchase.Charge, error) {
			return MapCreditPurchaseChargeFromDB(entity, input.Expands)
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
