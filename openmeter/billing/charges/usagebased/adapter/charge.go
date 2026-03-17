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

var _ usagebased.ChargeAdapter = (*adapter)(nil)

func (a *adapter) UpdateStatus(ctx context.Context, input usagebased.UpdateStatusInput) (usagebased.ChargeBase, error) {
	if err := input.Validate(); err != nil {
		return usagebased.ChargeBase{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.ChargeBase, error) {
		metaStatus, err := input.NewStatus.ToMetaChargeStatus()
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		updatedMeta, err := tx.metaAdapter.UpdateStatus(ctx, meta.UpdateStatusInput{
			ChargeID:     input.Charge.GetChargeID(),
			Status:       metaStatus,
			AdvanceAfter: input.Charge.State.AdvanceAfter,
		})
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		dbUpdatedChargeBase, err := tx.db.ChargeUsageBased.UpdateOneID(input.Charge.ID).
			Where(dbchargeusagebased.NamespaceEQ(input.Charge.Namespace)).
			SetStatus(input.NewStatus).
			Save(ctx)
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		return MapChargeBaseFromDB(dbUpdatedChargeBase, updatedMeta), nil
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

		updatedMeta, err := tx.metaAdapter.UpdateStatus(ctx, meta.UpdateStatusInput{
			ChargeID:     charge.GetChargeID(),
			Status:       metaStatus,
			AdvanceAfter: charge.State.AdvanceAfter,
		})
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		update := tx.db.ChargeUsageBased.UpdateOneID(charge.ID).
			Where(dbchargeusagebased.NamespaceEQ(charge.Namespace)).
			SetDiscounts(&charge.Intent.Discounts).
			SetStatus(charge.Status).
			SetOrClearCurrentRealizationRunID(charge.State.CurrentRealizationRunID)

		dbUpdatedChargeBase, err := update.Save(ctx)
		if err != nil {
			return usagebased.ChargeBase{}, err
		}

		return MapChargeBaseFromDB(dbUpdatedChargeBase, updatedMeta), nil
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
			chargeBase := MapChargeBaseFromDB(entity, chargeMetas[idx])
			out = append(out, usagebased.Charge{
				ChargeBase: chargeBase,
			})
		}
		return out, nil
	})
}

func (a *adapter) GetByMetas(ctx context.Context, input usagebased.GetByMetasInput) ([]usagebased.Charge, error) {
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
			chargeBase := MapChargeBaseFromDB(entity, input.Charges[idx])

			var realizations usagebased.RealizationRuns
			if input.Expands.Has(meta.ExpandRealizations) {
				realizations, err = MapRealizationRunsFromDB(entity)
				if err != nil {
					return nil, err
				}
			}

			entitiesMapped = append(entitiesMapped, usagebased.Charge{
				ChargeBase:   chargeBase,
				Realizations: realizations,
			})
		}

		entitiesByID := lo.KeyBy(entitiesMapped, func(charge usagebased.Charge) string {
			return charge.ID
		})

		var errs []error
		out := make([]usagebased.Charge, 0, len(input.Charges))
		for _, charge := range input.Charges {
			mapped, ok := entitiesByID[charge.ID]
			if !ok {
				errs = append(errs, fmt.Errorf("charge not found: %s", charge.ID))
				continue
			}

			out = append(out, mapped)
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

func (a *adapter) GetByID(ctx context.Context, input usagebased.GetByIDInput) (usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return usagebased.Charge{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.Charge, error) {
		metas, err := tx.metaAdapter.GetByIDs(ctx, meta.GetByIDsInput{
			Namespace: input.ChargeID.Namespace,
			ChargeIDs: []string{input.ChargeID.ID},
		})
		if err != nil {
			return usagebased.Charge{}, err
		}
		if len(metas) != 1 {
			return usagebased.Charge{}, fmt.Errorf("expected 1 meta, got %d", len(metas))
		}

		charges, err := a.GetByMetas(ctx, usagebased.GetByMetasInput{
			Namespace: input.ChargeID.Namespace,
			Charges:   meta.Charges{metas[0]},
			Expands:   input.Expands,
		})
		if err != nil {
			return usagebased.Charge{}, err
		}
		if len(charges) != 1 {
			return usagebased.Charge{}, fmt.Errorf("expected 1 charge, got %d", len(charges))
		}

		return charges[0], nil
	})
}

func (a *adapter) buildCreateUsageBasedCharge(ctx context.Context, chargeMeta meta.Charge, intent usagebased.Intent) (*db.ChargeUsageBasedCreate, error) {
	return a.db.ChargeUsageBased.Create().
		SetNamespace(chargeMeta.Namespace).
		SetID(chargeMeta.ID).
		SetChargeID(chargeMeta.ID).
		SetDiscounts(&intent.Discounts).
		SetPrice(&intent.Price).
		SetStatus(usagebased.Status(chargeMeta.Status)).
		SetFeatureKey(intent.FeatureKey).
		SetInvoiceAt(intent.InvoiceAt.In(time.UTC)).
		SetSettlementMode(intent.SettlementMode), nil
}
