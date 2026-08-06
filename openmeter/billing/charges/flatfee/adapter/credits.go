package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeeruncreditallocations"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeerunoveragecreditallocations"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ flatfee.ChargeCreditAllocationAdapter = (*adapter)(nil)

func (a *adapter) CreateChargeCurrencyCreditRealizations(ctx context.Context, input flatfee.CreateCreditRealizationsInput) (creditrealization.Realizations, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		correctionTargetIDs := input.CreditRealizations.CorrectionTargetIDs()
		if len(correctionTargetIDs) > 0 {
			persistedTargetIDs, err := tx.db.ChargeFlatFeeRunCreditAllocations.Query().
				Where(
					chargeflatfeeruncreditallocations.NamespaceEQ(input.RunID.Namespace),
					chargeflatfeeruncreditallocations.RunIDEQ(input.RunID.ID),
					chargeflatfeeruncreditallocations.IDIn(correctionTargetIDs...),
					chargeflatfeeruncreditallocations.TypeEQ(creditrealization.TypeAllocation),
				).
				IDs(ctx)
			if err != nil {
				return nil, fmt.Errorf("querying charge currency correction targets for flat fee realization run [run_id=%s]: %w", input.RunID.ID, err)
			}

			if err := validateCorrectionTargets(input.RunID.ID, correctionTargetIDs, persistedTargetIDs); err != nil {
				return nil, err
			}
		}

		dbEntities, err := tx.db.ChargeFlatFeeRunCreditAllocations.CreateBulk(
			lo.Map(input.CreditRealizations, func(realization creditrealization.CreateInput, idx int) *db.ChargeFlatFeeRunCreditAllocationsCreate {
				create := tx.db.ChargeFlatFeeRunCreditAllocations.Create().
					SetRunID(input.RunID.ID)
				create = creditrealization.Create(create, input.RunID.Namespace, idx, realization)

				return create
			})...,
		).Save(ctx)
		if err != nil {
			return creditrealization.Realizations{}, err
		}

		return creditrealization.FromDBRealizations(dbEntities), nil
	})
}

func (a *adapter) CreateFiatOverageCreditRealizations(ctx context.Context, input flatfee.CreateCreditRealizationsInput) (creditrealization.Realizations, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		correctionTargetIDs := input.CreditRealizations.CorrectionTargetIDs()
		if len(correctionTargetIDs) > 0 {
			persistedTargetIDs, err := tx.db.ChargeFlatFeeRunOverageCreditAllocations.Query().
				Where(
					chargeflatfeerunoveragecreditallocations.NamespaceEQ(input.RunID.Namespace),
					chargeflatfeerunoveragecreditallocations.RunIDEQ(input.RunID.ID),
					chargeflatfeerunoveragecreditallocations.IDIn(correctionTargetIDs...),
					chargeflatfeerunoveragecreditallocations.TypeEQ(creditrealization.TypeAllocation),
				).
				IDs(ctx)
			if err != nil {
				return nil, fmt.Errorf("querying fiat overage correction targets for flat fee realization run [run_id=%s]: %w", input.RunID.ID, err)
			}

			if err := validateCorrectionTargets(input.RunID.ID, correctionTargetIDs, persistedTargetIDs); err != nil {
				return nil, err
			}
		}

		dbEntities, err := tx.db.ChargeFlatFeeRunOverageCreditAllocations.CreateBulk(
			lo.Map(input.CreditRealizations, func(realization creditrealization.CreateInput, idx int) *db.ChargeFlatFeeRunOverageCreditAllocationsCreate {
				create := tx.db.ChargeFlatFeeRunOverageCreditAllocations.Create().
					SetRunID(input.RunID.ID)
				create = creditrealization.Create(create, input.RunID.Namespace, idx, realization)

				return create
			})...,
		).Save(ctx)
		if err != nil {
			return creditrealization.Realizations{}, err
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
			"correction targets must be persisted allocations in the same flat fee realization run [run_id=%s target_ids=%v]",
			runID,
			missingTargetIDs,
		))
	}

	return nil
}
