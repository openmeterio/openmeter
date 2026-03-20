package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	dbchargecreditpurchase "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ creditpurchase.Adapter = (*adapter)(nil)

func (a *adapter) UpdateCharge(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	if err := charge.Validate(); err != nil {
		return creditpurchase.Charge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.Charge, error) {
		updatedMeta, err := tx.metaAdapter.UpdateStatus(ctx, meta.UpdateStatusInput{
			ChargeID: charge.GetChargeID(),
			Status:   charge.Status,
		})
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		creditPurchaseUpdate := tx.db.ChargeCreditPurchase.UpdateOneID(charge.ID).
			Where(dbchargecreditpurchase.NamespaceEQ(charge.Namespace)).
			SetCreditAmount(charge.Intent.CreditAmount).
			SetSettlement(charge.Intent.Settlement)

		if charge.State.CreditGrantRealization != nil {
			creditPurchaseUpdate = creditPurchaseUpdate.
				SetCreditGrantTransactionGroupID(charge.State.CreditGrantRealization.TransactionGroupID).
				SetCreditGrantedAt(charge.State.CreditGrantRealization.Time.In(time.UTC))
		}

		dbCreditPurchase, err := creditPurchaseUpdate.Save(ctx)
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		mapped, err := MapCreditPurchaseChargeFromDB(updatedMeta, dbCreditPurchase, meta.ExpandNone)
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		mapped.State.ExternalPaymentSettlement = charge.State.ExternalPaymentSettlement

		return mapped, nil
	})
}

func (a *adapter) CreateCharge(ctx context.Context, in creditpurchase.CreateChargeInput) (creditpurchase.Charge, error) {
	if err := in.Validate(); err != nil {
		return creditpurchase.Charge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.Charge, error) {
		chargeMetas, err := tx.metaAdapter.Create(ctx, meta.CreateInput{
			Namespace: in.Namespace,
			Intents: []meta.IntentCreate{
				{
					Intent: in.Intent.Intent,
					Type:   meta.ChargeTypeCreditPurchase,
				},
			},
		})
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		if len(chargeMetas) != 1 {
			return creditpurchase.Charge{}, fmt.Errorf("expected 1 charge meta, got %d", len(chargeMetas))
		}

		chargeMeta := chargeMetas[0]

		create := tx.db.ChargeCreditPurchase.Create().
			SetNamespace(in.Namespace).
			SetID(chargeMeta.ID).
			SetChargeID(chargeMeta.ID).
			SetCreditAmount(in.Intent.CreditAmount).
			SetSettlement(in.Intent.Settlement)

			// Note: given that the ID (PK) matches the ChargeID edge, we cannot use the Create method directly
			// as it will cause duplicate id inserts. CreateBulk deduplicates the IDs, so it just works fine (sic).
		dbCreditPurchases, err := tx.db.ChargeCreditPurchase.CreateBulk(create).Save(ctx)
		if err != nil {
			return creditpurchase.Charge{}, err
		}
		if len(dbCreditPurchases) != 1 {
			return creditpurchase.Charge{}, fmt.Errorf("expected 1 credit purchase, got %d", len(dbCreditPurchases))
		}
		dbCreditPurchase := dbCreditPurchases[0]

		return MapCreditPurchaseChargeFromDB(chargeMeta, dbCreditPurchase, meta.ExpandNone)
	})
}

func (a *adapter) GetByMetas(ctx context.Context, input creditpurchase.GetByMetasInput) ([]creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]creditpurchase.Charge, error) {
		query := tx.db.ChargeCreditPurchase.Query().
			Where(dbchargecreditpurchase.Namespace(input.Namespace)).
			Where(dbchargecreditpurchase.IDIn(lo.Map(input.Charges, func(charge meta.Charge, idx int) string {
				return charge.ID
			})...))

		if input.Expands.Has(meta.ExpandRealizations) {
			query = query.WithExternalPayment()
		}

		entities, err := query.All(ctx)
		if err != nil {
			return nil, err
		}

		entitiesMapped := make([]creditpurchase.Charge, 0, len(entities))
		for idx, entity := range entities {
			charge, err := MapCreditPurchaseChargeFromDB(input.Charges[idx], entity, input.Expands)
			if err != nil {
				return nil, err
			}
			entitiesMapped = append(entitiesMapped, charge)
		}

		entitiesByID := lo.GroupBy(entitiesMapped, func(charge creditpurchase.Charge) string {
			return charge.ID
		})

		var errs []error
		out := make([]creditpurchase.Charge, 0, len(input.Charges))
		for _, charge := range input.Charges {
			charges, ok := entitiesByID[charge.ID]
			if !ok {
				errs = append(errs, fmt.Errorf("charge not found: %s", charge.ID))
				continue
			}

			out = append(out, charges[0])
		}

		if len(errs) > 0 {
			return nil, errors.Join(errs...)
		}

		return out, nil
	})
}
