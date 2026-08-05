package adapter

import (
	"context"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeeruncreditallocations"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeerunoveragecreditallocations"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *FlatFeeRealizationRunAdapterSuite) TestCreditRealizationCorrectionsTargetSameRunAllocations() {
	type realizationScope struct {
		name   string
		create func(context.Context, flatfee.CreateCreditRealizationsInput) (creditrealization.Realizations, error)
		count  func(context.Context, flatfee.RealizationRunID) (int, error)
	}

	scopes := []realizationScope{
		{
			name:   "charge_currency",
			create: s.adapter.CreateChargeCurrencyCreditRealizations,
			count: func(ctx context.Context, runID flatfee.RealizationRunID) (int, error) {
				return s.dbClient.ChargeFlatFeeRunCreditAllocations.Query().
					Where(
						chargeflatfeeruncreditallocations.NamespaceEQ(runID.Namespace),
						chargeflatfeeruncreditallocations.RunIDEQ(runID.ID),
					).
					Count(ctx)
			},
		},
		{
			name:   "fiat_overage",
			create: s.adapter.CreateFiatOverageCreditRealizations,
			count: func(ctx context.Context, runID flatfee.RealizationRunID) (int, error) {
				return s.dbClient.ChargeFlatFeeRunOverageCreditAllocations.Query().
					Where(
						chargeflatfeerunoveragecreditallocations.NamespaceEQ(runID.Namespace),
						chargeflatfeerunoveragecreditallocations.RunIDEQ(runID.ID),
					).
					Count(ctx)
			},
		},
	}

	for _, scope := range scopes {
		s.Run(scope.name, func() {
			ctx := s.T().Context()
			namespace := "flatfee-credit-realization-" + scope.name

			// Given two realization runs with persisted allocations in the same monetary domain.
			run, servicePeriod := s.createFlatFeeChargeWithRun(namespace)
			otherRun, otherServicePeriod := s.createFlatFeeChargeWithRun(namespace)

			allocations, err := scope.create(ctx, flatfee.CreateCreditRealizationsInput{
				RunID: run,
				CreditRealizations: creditrealization.CreateInputs{
					newFlatFeeCreditRealization(servicePeriod, creditrealization.TypeAllocation, 10, nil),
				},
			})
			s.Require().NoError(err)
			s.Require().Len(allocations, 1)

			otherAllocations, err := scope.create(ctx, flatfee.CreateCreditRealizationsInput{
				RunID: otherRun,
				CreditRealizations: creditrealization.CreateInputs{
					newFlatFeeCreditRealization(otherServicePeriod, creditrealization.TypeAllocation, 10, nil),
				},
			})
			s.Require().NoError(err)
			s.Require().Len(otherAllocations, 1)

			// When a correction targets an allocation persisted in its own run.
			corrections, err := scope.create(ctx, flatfee.CreateCreditRealizationsInput{
				RunID: run,
				CreditRealizations: creditrealization.CreateInputs{
					newFlatFeeCreditRealization(servicePeriod, creditrealization.TypeCorrection, -1, lo.ToPtr(allocations[0].ID)),
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
					input := flatfee.CreateCreditRealizationsInput{
						RunID: run,
						CreditRealizations: creditrealization.CreateInputs{
							newFlatFeeCreditRealization(servicePeriod, creditrealization.TypeCorrection, -1, lo.ToPtr(invalidTarget.targetID)),
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
			countBefore, err := scope.count(ctx, run)
			s.Require().NoError(err)
			mixedBatch := flatfee.CreateCreditRealizationsInput{
				RunID: run,
				CreditRealizations: creditrealization.CreateInputs{
					newFlatFeeCreditRealization(servicePeriod, creditrealization.TypeCorrection, -1, lo.ToPtr(allocations[0].ID)),
					newFlatFeeCreditRealization(servicePeriod, creditrealization.TypeCorrection, -1, lo.ToPtr(otherAllocations[0].ID)),
				},
			}

			// When the mixed batch is persisted.
			_, err = scope.create(ctx, mixedBatch)

			// Then the entire batch is rejected without persisting the valid correction.
			s.Require().Error(err)
			countAfter, countErr := scope.count(ctx, run)
			s.Require().NoError(countErr)
			s.Equal(countBefore, countAfter)
		})
	}
}

func (s *FlatFeeRealizationRunAdapterSuite) createFlatFeeChargeWithRun(namespace string) (flatfee.RealizationRunID, timeutil.ClosedPeriod) {
	s.T().Helper()

	customerID := s.createCustomer(namespace)
	taxCodeID := s.taxCodeEnv.CreateTaxCode(s.T(), namespace).ID
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	createdCharges, err := s.adapter.CreateCharges(s.T().Context(), flatfee.CreateChargesInput{
		Namespace: namespace,
		Intents: []flatfee.IntentWithInitialStatus{
			{
				Intent: flatfee.Intent{
					Intent: chargesmeta.Intent{
						ManagedBy:  billing.SubscriptionManagedLine,
						CustomerID: customerID,
						Currency:   currenciestestutils.NewFiatCurrency(s.T(), "USD"),
						TaxConfig: productcatalog.TaxCodeConfig{
							TaxCodeID: taxCodeID,
						},
					},
					IntentMutableFields: flatfee.IntentMutableFields{
						IntentMutableFields: chargesmeta.IntentMutableFields{
							Name:              "flat-fee-charge",
							ServicePeriod:     servicePeriod,
							FullServicePeriod: servicePeriod,
							BillingPeriod:     servicePeriod,
						},
						InvoiceAt:             servicePeriod.To,
						PaymentTerm:           productcatalog.InAdvancePaymentTerm,
						AmountBeforeProration: alpacadecimal.NewFromInt(10),
						ProRating: productcatalog.ProRatingConfig{
							Enabled: false,
							Mode:    productcatalog.ProRatingModeProratePrices,
						},
					},
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				},
				InitialStatus:        flatfee.StatusCreated,
				AmountAfterProration: alpacadecimal.NewFromInt(10),
			},
		},
	})
	s.Require().NoError(err)
	s.Require().Len(createdCharges, 1)

	run, err := s.adapter.CreateCurrentRun(s.T().Context(), flatfee.CreateCurrentRunInput{
		Charge:               createdCharges[0].ChargeBase,
		ServicePeriod:        servicePeriod,
		AmountAfterProration: alpacadecimal.NewFromInt(10),
	})
	s.Require().NoError(err)

	return run.ID, servicePeriod
}

func newFlatFeeCreditRealization(servicePeriod timeutil.ClosedPeriod, realizationType creditrealization.Type, amount int64, correctsRealizationID *string) creditrealization.CreateInput {
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
