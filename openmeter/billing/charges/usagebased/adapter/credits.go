package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedruncreditallocations"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedrunoveragecreditallocations"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ usagebased.RealizationRunCreditAllocationAdapter = (*adapter)(nil)

func (a *adapter) CreateChargeCurrencyCreditRealizations(ctx context.Context, input usagebased.CreateCreditRealizationsInput) (creditrealization.Realizations, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		correctionTargetIDs := input.CreditRealizations.CorrectionTargetIDs()
		if len(correctionTargetIDs) > 0 {
			persistedTargetIDs, err := tx.db.ChargeUsageBasedRunCreditAllocations.Query().
				Where(
					chargeusagebasedruncreditallocations.NamespaceEQ(input.RunID.Namespace),
					chargeusagebasedruncreditallocations.RunIDEQ(input.RunID.ID),
					chargeusagebasedruncreditallocations.IDIn(correctionTargetIDs...),
					chargeusagebasedruncreditallocations.TypeEQ(creditrealization.TypeAllocation),
				).
				IDs(ctx)
			if err != nil {
				return nil, fmt.Errorf("querying charge currency correction targets for usage based realization run [run_id=%s]: %w", input.RunID.ID, err)
			}

			if err := validateCorrectionTargets(input.RunID.ID, correctionTargetIDs, persistedTargetIDs); err != nil {
				return nil, err
			}
		}

		creates := lo.Map(input.CreditRealizations, func(realization creditrealization.CreateInput, idx int) *entdb.ChargeUsageBasedRunCreditAllocationsCreate {
			create := tx.db.ChargeUsageBasedRunCreditAllocations.Create().
				SetRunID(input.RunID.ID).
				SetNamespace(input.RunID.Namespace)

			create = creditrealization.Create(create, input.RunID.Namespace, idx, realization)

			return create
		})

		dbEntities, err := tx.db.ChargeUsageBasedRunCreditAllocations.CreateBulk(creates...).Save(ctx)
		if err != nil {
			return nil, err
		}

		return creditrealization.FromDBRealizations(dbEntities), nil
	})
}

func (a *adapter) CreateFiatOverageCreditRealizations(ctx context.Context, input usagebased.CreateCreditRealizationsInput) (creditrealization.Realizations, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		correctionTargetIDs := input.CreditRealizations.CorrectionTargetIDs()
		if len(correctionTargetIDs) > 0 {
			persistedTargetIDs, err := tx.db.ChargeUsageBasedRunOverageCreditAllocations.Query().
				Where(
					chargeusagebasedrunoveragecreditallocations.NamespaceEQ(input.RunID.Namespace),
					chargeusagebasedrunoveragecreditallocations.RunIDEQ(input.RunID.ID),
					chargeusagebasedrunoveragecreditallocations.IDIn(correctionTargetIDs...),
					chargeusagebasedrunoveragecreditallocations.TypeEQ(creditrealization.TypeAllocation),
				).
				IDs(ctx)
			if err != nil {
				return nil, fmt.Errorf("querying fiat overage correction targets for usage based realization run [run_id=%s]: %w", input.RunID.ID, err)
			}

			if err := validateCorrectionTargets(input.RunID.ID, correctionTargetIDs, persistedTargetIDs); err != nil {
				return nil, err
			}
		}

		creates := lo.Map(input.CreditRealizations, func(realization creditrealization.CreateInput, idx int) *entdb.ChargeUsageBasedRunOverageCreditAllocationsCreate {
			create := tx.db.ChargeUsageBasedRunOverageCreditAllocations.Create().
				SetRunID(input.RunID.ID).
				SetNamespace(input.RunID.Namespace)

			create = creditrealization.Create(create, input.RunID.Namespace, idx, realization)

			return create
		})

		dbEntities, err := tx.db.ChargeUsageBasedRunOverageCreditAllocations.CreateBulk(creates...).Save(ctx)
		if err != nil {
			return nil, err
		}

		return creditrealization.FromDBRealizations(dbEntities), nil
	})
}

// validateCorrectionTargets enforces that corrections reference allocations in the same
// realization run. The self-referencing foreign key only guarantees that the target exists
// in this allocation table; scoping it to the run would require a composite foreign key over
// the target and run IDs. Ent cannot model that constraint directly, so implementing it in
// the database would require hand-maintained migration and schema-diff exclusions.
func validateCorrectionTargets(runID string, targetIDs, persistedTargetIDs []string) error {
	persistedTargets := make(map[string]struct{}, len(persistedTargetIDs))
	for _, targetID := range persistedTargetIDs {
		persistedTargets[targetID] = struct{}{}
	}

	missingTargetIDs := make([]string, 0)
	for _, targetID := range targetIDs {
		if _, ok := persistedTargets[targetID]; !ok {
			missingTargetIDs = append(missingTargetIDs, targetID)
		}
	}

	if len(missingTargetIDs) > 0 {
		return models.NewGenericValidationError(fmt.Errorf(
			"correction targets must be persisted allocations in the same usage based realization run [run_id=%s target_ids=%v]",
			runID,
			missingTargetIDs,
		))
	}

	return nil
}
