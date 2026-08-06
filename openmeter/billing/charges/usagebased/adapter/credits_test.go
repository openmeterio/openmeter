package adapter

import (
	"context"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedruncreditallocations"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedrunoveragecreditallocations"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *DetailedLineAdapterSuite) TestCreditRealizationCorrectionsTargetSameRunAllocations() {
	type realizationScope struct {
		name   string
		create func(context.Context, usagebased.CreateCreditRealizationsInput) (creditrealization.Realizations, error)
		count  func(context.Context, usagebased.RealizationRunID) (int, error)
	}

	scopes := []realizationScope{
		{
			name:   "charge_currency",
			create: s.adapter.CreateChargeCurrencyCreditRealizations,
			count: func(ctx context.Context, runID usagebased.RealizationRunID) (int, error) {
				return s.dbClient.ChargeUsageBasedRunCreditAllocations.Query().
					Where(
						chargeusagebasedruncreditallocations.NamespaceEQ(runID.Namespace),
						chargeusagebasedruncreditallocations.RunIDEQ(runID.ID),
					).
					Count(ctx)
			},
		},
		{
			name:   "fiat_overage",
			create: s.adapter.CreateFiatOverageCreditRealizations,
			count: func(ctx context.Context, runID usagebased.RealizationRunID) (int, error) {
				return s.dbClient.ChargeUsageBasedRunOverageCreditAllocations.Query().
					Where(
						chargeusagebasedrunoveragecreditallocations.NamespaceEQ(runID.Namespace),
						chargeusagebasedrunoveragecreditallocations.RunIDEQ(runID.ID),
					).
					Count(ctx)
			},
		},
	}

	for _, scope := range scopes {
		s.Run(scope.name, func() {
			ctx := s.T().Context()
			namespace := "usagebased-credit-realization-" + scope.name

			// Given two realization runs with persisted allocations in the same monetary domain.
			_, run, servicePeriod := s.createChargeWithRun(namespace)
			_, otherRun, otherServicePeriod := s.createChargeWithRun(namespace)

			allocations, err := scope.create(ctx, usagebased.CreateCreditRealizationsInput{
				RunID: run.ID,
				CreditRealizations: creditrealization.CreateInputs{
					newUsageBasedCreditRealization(servicePeriod, creditrealization.TypeAllocation, 10, nil),
				},
			})
			s.Require().NoError(err)
			s.Require().Len(allocations, 1)

			otherAllocations, err := scope.create(ctx, usagebased.CreateCreditRealizationsInput{
				RunID: otherRun.ID,
				CreditRealizations: creditrealization.CreateInputs{
					newUsageBasedCreditRealization(otherServicePeriod, creditrealization.TypeAllocation, 10, nil),
				},
			})
			s.Require().NoError(err)
			s.Require().Len(otherAllocations, 1)

			// When a correction targets an allocation persisted in its own run.
			corrections, err := scope.create(ctx, usagebased.CreateCreditRealizationsInput{
				RunID: run.ID,
				CreditRealizations: creditrealization.CreateInputs{
					newUsageBasedCreditRealization(servicePeriod, creditrealization.TypeCorrection, -1, lo.ToPtr(allocations[0].ID)),
				},
			})

			// Then the correction is persisted.
			s.Require().NoError(err)
			s.Require().Len(corrections, 1)
			s.Equal(allocations[0].ID, *corrections[0].CorrectsRealizationID)

			invalidTargets := []struct {
				name     string
				targetID string
			}{
				{name: "another run", targetID: otherAllocations[0].ID},
				{name: "unknown allocation", targetID: ulid.Make().String()},
				{name: "another correction", targetID: corrections[0].ID},
			}

			for _, invalidTarget := range invalidTargets {
				s.Run(invalidTarget.name, func() {
					// Given a correction target that is not an allocation in the current run.
					input := usagebased.CreateCreditRealizationsInput{
						RunID: run.ID,
						CreditRealizations: creditrealization.CreateInputs{
							newUsageBasedCreditRealization(servicePeriod, creditrealization.TypeCorrection, -1, lo.ToPtr(invalidTarget.targetID)),
						},
					}

					// When the correction is persisted.
					_, err := scope.create(ctx, input)

					// Then the adapter rejects the target as a validation error.
					s.Require().Error(err)
					s.True(models.IsGenericValidationError(err))
					s.Contains(err.Error(), invalidTarget.targetID)
				})
			}

			// Given a correction batch containing both same-run and cross-run targets.
			countBefore, err := scope.count(ctx, run.ID)
			s.Require().NoError(err)
			mixedBatch := usagebased.CreateCreditRealizationsInput{
				RunID: run.ID,
				CreditRealizations: creditrealization.CreateInputs{
					newUsageBasedCreditRealization(servicePeriod, creditrealization.TypeCorrection, -1, lo.ToPtr(allocations[0].ID)),
					newUsageBasedCreditRealization(servicePeriod, creditrealization.TypeCorrection, -1, lo.ToPtr(otherAllocations[0].ID)),
				},
			}

			// When the mixed batch is persisted.
			_, err = scope.create(ctx, mixedBatch)

			// Then the entire batch is rejected without persisting the valid correction.
			s.Require().Error(err)
			countAfter, countErr := scope.count(ctx, run.ID)
			s.Require().NoError(countErr)
			s.Equal(countBefore, countAfter)
		})
	}
}

func newUsageBasedCreditRealization(servicePeriod timeutil.ClosedPeriod, realizationType creditrealization.Type, amount int64, correctsRealizationID *string) creditrealization.CreateInput {
	return creditrealization.CreateInput{
		ServicePeriod: servicePeriod,
		LedgerTransaction: ledgertransaction.GroupReference{
			TransactionGroupID: ulid.Make().String(),
		},
		Amount:                alpacadecimal.NewFromInt(amount),
		Type:                  realizationType,
		CorrectsRealizationID: correctsRealizationID,
	}
}
