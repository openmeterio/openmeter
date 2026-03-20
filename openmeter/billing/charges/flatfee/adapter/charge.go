package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
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

		_, err := tx.metaAdapter.UpdateStatus(ctx, meta.UpdateStatusInput{
			ChargeID: charge.GetChargeID(),
			Status:   charge.Status,
		})
		if err != nil {
			return err
		}

		var discounts *productcatalog.Discounts
		if intent.PercentageDiscounts != nil {
			discounts = &productcatalog.Discounts{Percentage: intent.PercentageDiscounts}
		}

		proRating, err := proRatingConfigToDB(intent.ProRating)
		if err != nil {
			return err
		}

		create := tx.db.ChargeFlatFee.UpdateOneID(charge.ID).
			Where(dbchargeflatfee.NamespaceEQ(charge.Namespace)).
			SetPaymentTerm(intent.PaymentTerm).
			SetInvoiceAt(intent.InvoiceAt.In(time.UTC)).
			SetDiscounts(discounts).
			SetProRating(proRating).
			SetAmountBeforeProration(intent.AmountBeforeProration).
			SetAmountAfterProration(intent.AmountAfterProration)

		_, err = create.Save(ctx)
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
		chargeMetas, err := tx.metaAdapter.Create(ctx, meta.CreateInput{
			Namespace: in.Namespace,
			Intents: slicesx.Map(in.Intents, func(intent flatfee.IntentWithInitialStatus) meta.IntentCreate {
				return meta.IntentCreate{
					Intent:        intent.Intent.Intent,
					Type:          meta.ChargeTypeFlatFee,
					InitialStatus: intent.InitialStatus,
				}
			}),
		})
		if err != nil {
			return nil, err
		}

		if len(chargeMetas) != len(in.Intents) {
			return nil, fmt.Errorf("expected %d charge metas, got %d", len(in.Intents), len(chargeMetas))
		}

		creates := make([]*db.ChargeFlatFeeCreate, 0, len(chargeMetas))
		for idx, chargeMeta := range chargeMetas {
			create, err := tx.buildCreateFlatFeeCharge(ctx, chargeMeta, in.Intents[idx].Intent)
			if err != nil {
				return nil, err
			}

			creates = append(creates, create)
		}

		entities, err := tx.db.ChargeFlatFee.CreateBulk(creates...).Save(ctx)
		if err != nil {
			return nil, err
		}

		out := make([]flatfee.Charge, 0, len(entities))
		for idx, entity := range entities {
			charge, err := MapChargeFlatFeeFromDB(entity, chargeMetas[idx], meta.ExpandNone)
			if err != nil {
				return nil, err
			}
			out = append(out, charge)
		}
		return out, nil
	})
}

func (a *adapter) GetByMetas(ctx context.Context, input flatfee.GetByMetasInput) ([]flatfee.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]flatfee.Charge, error) {
		query := tx.db.ChargeFlatFee.Query().
			Where(dbchargeflatfee.Namespace(input.Namespace)).
			Where(dbchargeflatfee.IDIn(
				lo.Map(input.Charges, func(charge meta.Charge, idx int) string {
					return charge.ID
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

		entitiesMapped := make([]flatfee.Charge, 0, len(entities))
		for idx, entity := range entities {
			charge, err := MapChargeFlatFeeFromDB(entity, input.Charges[idx], input.Expands)
			if err != nil {
				return nil, err
			}
			entitiesMapped = append(entitiesMapped, charge)
		}

		entitiesByID := lo.GroupBy(entitiesMapped, func(charge flatfee.Charge) string {
			return charge.ID
		})

		var errs []error
		out := make([]flatfee.Charge, 0, len(input.Charges))
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

func (a *adapter) buildCreateFlatFeeCharge(ctx context.Context, chargeMeta meta.Charge, intent flatfee.Intent) (*db.ChargeFlatFeeCreate, error) {
	var discounts *productcatalog.Discounts
	if intent.PercentageDiscounts != nil {
		discounts = &productcatalog.Discounts{Percentage: intent.PercentageDiscounts}
	}

	proRating, err := proRatingConfigToDB(intent.ProRating)
	if err != nil {
		return nil, err
	}

	create := a.db.ChargeFlatFee.Create().
		SetID(chargeMeta.ID).
		SetChargeID(chargeMeta.ID).
		SetNamespace(chargeMeta.Namespace).
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

	return create, nil
}
