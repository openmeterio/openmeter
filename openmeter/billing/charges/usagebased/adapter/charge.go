package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeusagebased "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebased"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (a *adapter) UpdateCharge(ctx context.Context, charge usagebased.Charge) error {
	if err := charge.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		_, err := tx.metaAdapter.UpdateStatus(ctx, meta.UpdateStatusInput{
			ChargeID: charge.GetChargeID(),
			Status:   charge.Status,
		})
		if err != nil {
			return err
		}

		_, err = tx.db.ChargeUsageBased.UpdateOneID(charge.ID).
			Where(dbchargeusagebased.NamespaceEQ(charge.Namespace)).
			SetDiscounts(&charge.Intent.Discounts).
			Save(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

func (a *adapter) CreateCharges(ctx context.Context, in usagebased.CreateInput) ([]usagebased.Charge, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]usagebased.Charge, error) {
		chargeMetas, err := tx.metaAdapter.Create(ctx, meta.CreateInput{
			Namespace: in.Namespace,
			Intents: slicesx.Map(in.Intents, func(intent usagebased.Intent) meta.IntentCreate {
				return meta.IntentCreate{
					Intent: intent.Intent,
					Type:   meta.ChargeTypeUsageBased,
				}
			}),
		})
		if err != nil {
			return nil, err
		}

		if len(chargeMetas) != len(in.Intents) {
			return nil, fmt.Errorf("expected %d charge metas, got %d", len(in.Intents), len(chargeMetas))
		}

		creates := make([]*db.ChargeUsageBasedCreate, 0, len(chargeMetas))
		for idx, chargeMeta := range chargeMetas {
			create, err := tx.buildCreateUsageBasedCharge(ctx, chargeMeta, in.Intents[idx])
			if err != nil {
				return nil, err
			}
			creates = append(creates, create)
		}
		entities, err := tx.db.ChargeUsageBased.CreateBulk(creates...).Save(ctx)
		if err != nil {
			return nil, err
		}

		out := make([]usagebased.Charge, 0, len(entities))
		for idx, entity := range entities {
			charge, err := MapUsageBasedChargeFromDB(entity, chargeMetas[idx], meta.ExpandNone)
			if err != nil {
				return nil, err
			}
			out = append(out, charge)
		}
		return out, nil
	})
}

func (a *adapter) GetByIDs(ctx context.Context, input usagebased.GetByIDsInput) ([]usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]usagebased.Charge, error) {
		query := tx.db.ChargeUsageBased.Query().
			Where(dbchargeusagebased.Namespace(input.Namespace)).
			Where(dbchargeusagebased.IDIn(
				lo.Map(input.Charges, func(charge meta.Charge, idx int) string {
					return charge.ID
				})...,
			))

		if input.Expands.Has(meta.ExpandRealizations) {
			query = query.WithRuns(
				func(runs *db.ChargeUsageBasedRunsQuery) {
					runs.WithCreditAllocations().
						WithInvoicedUsage().
						WithPayment()
				},
			)
		}

		entities, err := query.All(ctx)
		if err != nil {
			return nil, err
		}

		entitiesMapped := make([]usagebased.Charge, 0, len(entities))
		for idx, entity := range entities {
			charge, err := MapUsageBasedChargeFromDB(entity, input.Charges[idx], input.Expands)
			if err != nil {
				return nil, err
			}
			entitiesMapped = append(entitiesMapped, charge)
		}

		entitiesByID := lo.GroupBy(entitiesMapped, func(charge usagebased.Charge) string {
			return charge.ID
		})

		var errs []error
		out := make([]usagebased.Charge, 0, len(input.Charges))
		for _, charge := range input.Charges {
			charges, ok := entitiesByID[charge.ID]
			if !ok {
				errs = append(errs, fmt.Errorf("charge not found: %s", charge.ID))
				continue
			}

			out = append(out, charges[0])
		}

		if len(out) != len(input.Charges) {
			return nil, fmt.Errorf("expected to fetch %d charges, got %d", len(input.Charges), len(out))
		}

		if len(errs) > 0 {
			return nil, errors.Join(errs...)
		}

		return out, nil
	})
}

func (a *adapter) buildCreateUsageBasedCharge(ctx context.Context, chargeMeta meta.Charge, intent usagebased.Intent) (*db.ChargeUsageBasedCreate, error) {
	return a.db.ChargeUsageBased.Create().
		SetNamespace(chargeMeta.Namespace).
		SetID(chargeMeta.ID).
		SetChargeID(chargeMeta.ID).
		SetDiscounts(&intent.Discounts).
		SetPrice(&intent.Price).
		SetFeatureKey(intent.FeatureKey).
		SetInvoiceAt(intent.InvoiceAt.In(time.UTC)).
		SetSettlementMode(intent.SettlementMode), nil
}
