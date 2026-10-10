package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	chargedetailedline "github.com/openmeterio/openmeter/openmeter/billing/charges/models/detailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestMapFlatFeeDetailedLinesPreservesDiscountSnapshots(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name      string
		discounts chargedetailedline.AmountDiscounts
	}{
		{
			name: "discount with rounding adjustment",
			discounts: chargedetailedline.AmountDiscounts{
				{
					ChildUniqueReferenceID: "discount-reference",
					Description:            lo.ToPtr("percentage discount"),
					Reason: billing.NewDiscountReasonFrom(billing.PercentageDiscount{
						PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)},
						CorrelationID:      "percentage-discount",
					}),
					Amount:         alpacadecimal.NewFromFloat(49.99),
					RoundingAmount: alpacadecimal.NewFromFloat(0.01),
				},
			},
		},
		{name: "historical line without breakdown"},
	} {
		t.Run(test.name, func(t *testing.T) {
			// given: a charge snapshot whose totals include a discount
			servicePeriod := timeutil.ClosedPeriod{
				From: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			}
			line := newFlatFeeStandardLineForTest(servicePeriod)
			snapshot := flatfee.DetailedLine{
				Base: stddetailedline.Base{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						ID:        "charge-detail-id",
						Namespace: line.Namespace,
						Name:      "flat fee",
						CreatedAt: servicePeriod.From,
						UpdatedAt: servicePeriod.From,
					}),
					ChildUniqueReferenceID: "flat-fee-reference",
					Category:               stddetailedline.CategoryRegular,
					PaymentTerm:            productcatalog.InAdvancePaymentTerm,
					ServicePeriod:          servicePeriod,
					PerUnitAmount:          alpacadecimal.NewFromInt(100),
					Quantity:               alpacadecimal.NewFromInt(1),
					Totals: totals.Totals{
						Amount:         alpacadecimal.NewFromInt(100),
						DiscountsTotal: alpacadecimal.NewFromInt(50),
						Total:          alpacadecimal.NewFromInt(50),
					},
				},
				AmountDiscounts: test.discounts.Clone(),
			}

			// when: the charge facts are mapped into invoice calculation output
			mapped, err := mapFlatFeeDetailedLines(line, flatfee.RealizationRun{
				DetailedLines: mo.Some(flatfee.DetailedLines{snapshot}),
			})
			require.NoError(t, err)

			// then: billing owns identities, while snapshot facts and totals are preserved
			require.Len(t, mapped, 1)
			detail := mapped[0]
			require.NoError(t, detail.Validate())
			require.Equal(t, line.InvoiceID, detail.InvoiceID)
			require.Equal(t, snapshot.ChildUniqueReferenceID, detail.ChildUniqueReferenceID)
			require.Equal(t, snapshot.Totals, detail.Totals)
			require.Empty(t, detail.ID)
			require.True(t, detail.CreatedAt.IsZero())
			require.True(t, detail.UpdatedAt.IsZero())
			require.Len(t, detail.AmountDiscounts, len(snapshot.AmountDiscounts))
			if len(snapshot.AmountDiscounts) == 0 {
				return
			}

			discount := detail.AmountDiscounts[0]
			require.Equal(t, models.ManagedModelWithID{}, discount.ManagedModelWithID)
			require.Equal(t, snapshot.AmountDiscounts[0].ChildUniqueReferenceID, lo.FromPtr(discount.ChildUniqueReferenceID))
			require.Equal(t, snapshot.AmountDiscounts[0].Description, discount.Description)
			require.Equal(t, snapshot.AmountDiscounts[0].Reason, discount.Reason)
			require.Equal(t, float64(49.99), discount.Amount.InexactFloat64())
			require.Equal(t, float64(0.01), discount.RoundingAmount.InexactFloat64())
			*discount.Description = "invoice-only description"
			*discount.ChildUniqueReferenceID = "invoice-only reference"
			require.Equal(t, "percentage discount", *snapshot.AmountDiscounts[0].Description)
			require.Equal(t, "discount-reference", snapshot.AmountDiscounts[0].ChildUniqueReferenceID)
		})
	}
}

func TestCalculateFiatOverageForRun(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	charge := newFlatFeeCustomCurrencyCreditThenInvoiceChargeForTest(t, servicePeriod)

	tests := []struct {
		name                      string
		runTotals                 totals.Totals
		noFiatTransactionRequired bool
		conversionFails           bool
		expectFiatOverage         float64
		expectOmitInvoiceLine     bool
	}{
		{
			name: "zero converted overage does not depend on transaction requirement",
			runTotals: totals.Totals{
				Amount:       alpacadecimal.NewFromInt(3),
				CreditsTotal: alpacadecimal.NewFromInt(3),
			},
			noFiatTransactionRequired: false,
			expectOmitInvoiceLine:     true,
		},
		{
			name: "positive converted overage is retained when no transaction is required",
			runTotals: totals.Totals{
				Amount: alpacadecimal.NewFromInt(3),
				Total:  alpacadecimal.NewFromInt(3),
			},
			noFiatTransactionRequired: true,
			expectFiatOverage:         6,
		},
		{
			name: "conversion failure is returned",
			runTotals: totals.Totals{
				Amount: alpacadecimal.NewFromInt(3),
				Total:  alpacadecimal.NewFromInt(3),
			},
			conversionFails: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given: a custom-currency run where converted overage and transaction requirement intentionally differ
			charge := charge
			if test.conversionFails {
				charge.State.ResolvedCostBasis = nil
			}

			run := newFlatFeeCustomCurrencyRunForTest(
				servicePeriod,
				test.runTotals,
				test.noFiatTransactionRequired,
			)

			// when: the run's fiat overage and invoice-line omission decision are calculated
			fiatOverage, err := calculateFiatOverageForRun(charge, run)

			// then: conversion errors are returned and successful conversion alone controls omission
			if test.conversionFails {
				require.ErrorContains(t, err, "resolved cost basis is required")
				require.False(t, fiatOverage.ShouldOmitInvoiceLine)

				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expectFiatOverage, fiatOverage.FiatOverage.InexactFloat64())
			require.Equal(t, test.expectOmitInvoiceLine, fiatOverage.ShouldOmitInvoiceLine)
		})
	}
}

func TestResolveFlatFeeCustomCurrencyFinalizationOverageReusesPreparedGrossAmount(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	charge := newFlatFeeCustomCurrencyCreditThenInvoiceChargeForTest(t, servicePeriod)
	run := newFlatFeeCustomCurrencyRunForTest(
		servicePeriod,
		totals.Totals{Amount: alpacadecimal.NewFromInt(3), Total: alpacadecimal.NewFromInt(3)},
		false,
	)
	run.AccruedUsage = &invoicedusage.AccruedUsage{
		ServicePeriod: servicePeriod,
		Totals: totals.Totals{
			Amount: alpacadecimal.NewFromInt(5),
			Total:  alpacadecimal.NewFromInt(5),
		},
	}

	// The current cost basis would convert the run to 6 USD. Finalization must
	// reuse the persisted 5 USD result instead of converting again.
	fiatOverage, err := resolveFiatOverageForLinePopulation(
		charge,
		run,
		standardLinePopulationStageInvoiceFinalizing,
	)
	require.NoError(t, err)
	require.Equal(t, float64(5), fiatOverage.FiatOverage.InexactFloat64())
}

func TestPopulateFlatFeeCustomCurrencyOverageLineDeletionUsesConvertedOverage(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	charge := newFlatFeeCustomCurrencyCreditThenInvoiceChargeForTest(t, servicePeriod)

	for _, stage := range []standardLinePopulationStage{
		standardLinePopulationStageGatheringPreview,
		standardLinePopulationStageCollectionCompleted,
	} {
		t.Run(string(stage), func(t *testing.T) {
			t.Run("zero converted overage deletes the line", func(t *testing.T) {
				// given: a zero converted overage whose transaction flag does not request payment skipping
				run := newFlatFeeCustomCurrencyRunForTest(
					servicePeriod,
					totals.Totals{
						Amount:       alpacadecimal.NewFromInt(3),
						CreditsTotal: alpacadecimal.NewFromInt(3),
					},
					false,
				)
				line := newFlatFeeStandardLineForTest(servicePeriod)

				// when: the overage line is populated at a deletion-capable stage
				err := populateFlatFeeStandardLineFromRun(line, populateFlatFeeStandardLineFromRunInput{
					Charge: charge,
					Run:    run,
					Stage:  stage,
				})

				// then: line omission follows the converted overage
				require.NoError(t, err)
				require.NotNil(t, line.DeletedAt)
			})

			t.Run("positive converted overage retains the line without a transaction", func(t *testing.T) {
				// given: a positive converted overage whose settlement does not require a fiat transaction
				run := newFlatFeeCustomCurrencyRunForTest(
					servicePeriod,
					totals.Totals{
						Amount: alpacadecimal.NewFromInt(3),
						Total:  alpacadecimal.NewFromInt(3),
					},
					true,
				)
				line := newFlatFeeStandardLineForTest(servicePeriod)

				// when: the overage line is populated at a deletion-capable stage
				err := populateFlatFeeStandardLineFromRun(line, populateFlatFeeStandardLineFromRunInput{
					Charge: charge,
					Run:    run,
					Stage:  stage,
				})

				// then: the positive billable line is retained independently of payment handling
				require.NoError(t, err)
				require.Nil(t, line.DeletedAt)
			})

			t.Run("conversion failure returns an error without deleting the line", func(t *testing.T) {
				// given: a custom-currency run whose resolved cost basis is unavailable
				charge := charge
				charge.State.ResolvedCostBasis = nil
				run := newFlatFeeCustomCurrencyRunForTest(
					servicePeriod,
					totals.Totals{
						Amount: alpacadecimal.NewFromInt(3),
						Total:  alpacadecimal.NewFromInt(3),
					},
					true,
				)
				line := newFlatFeeStandardLineForTest(servicePeriod)

				// when: the overage line is populated at a deletion-capable stage
				err := populateFlatFeeStandardLineFromRun(line, populateFlatFeeStandardLineFromRunInput{
					Charge: charge,
					Run:    run,
					Stage:  stage,
				})

				// then: conversion failure is exposed before the line can be deleted
				require.ErrorContains(t, err, "resolved cost basis is required")
				require.Nil(t, line.DeletedAt)
			})
		})
	}
}

func newFlatFeeCustomCurrencyCreditThenInvoiceChargeForTest(t testing.TB, servicePeriod timeutil.ClosedPeriod) flatfee.Charge {
	t.Helper()

	customCurrency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode("TOKENS").
		WithName("Tokens").
		WithPrecision(4).
		Build()
	require.NoError(t, err)

	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)
	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromInt(2),
	})
	costBasisID := "cost-basis-id"
	createdAt := servicePeriod.From

	return flatfee.Charge{
		ChargeBase: flatfee.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
				ManagedModel: models.ManagedModel{
					CreatedAt: createdAt,
					UpdatedAt: createdAt,
				},
				ID: "charge-id",
			},
			Intent: flatfee.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.SubscriptionManagedLine,
					CustomerID: "customer-id",
					Currency: currencies.Currency{
						NamespacedID: models.NamespacedID{
							Namespace: "namespace",
							ID:        "currency-id",
						},
						Currency: customCurrency,
					},
					TaxConfig: productcatalog.TaxCodeConfig{TaxCodeID: "tax-code-id"},
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "flat fee",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:             servicePeriod.To,
					PaymentTerm:           productcatalog.InArrearsPaymentTerm,
					AmountBeforeProration: alpacadecimal.NewFromInt(3),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				CostBasis:      &costBasisIntent,
			}.AsOverridableIntent(),
			Status: flatfee.StatusActiveRealizationProcessing,
			State: flatfee.State{
				AmountAfterProration: alpacadecimal.NewFromInt(3),
				CostBasisID:          &costBasisID,
				ResolvedCostBasis: &costbasis.State{
					CostBasis:  alpacadecimal.NewFromInt(2),
					ResolvedAt: servicePeriod.From,
				},
			},
		},
	}
}

func newFlatFeeCustomCurrencyRunForTest(
	servicePeriod timeutil.ClosedPeriod,
	runTotals totals.Totals,
	noFiatTransactionRequired bool,
) flatfee.RealizationRun {
	return flatfee.RealizationRun{
		RealizationRunBase: flatfee.RealizationRunBase{
			ID: flatfee.RealizationRunID{
				Namespace: "namespace",
				ID:        "run-id",
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: servicePeriod.From,
				UpdatedAt: servicePeriod.From,
			},
			Type:                      flatfee.RealizationRunTypeFinalRealization,
			InitialType:               flatfee.RealizationRunTypeFinalRealization,
			ServicePeriod:             servicePeriod,
			AmountAfterProration:      alpacadecimal.NewFromInt(3),
			Totals:                    runTotals,
			NoFiatTransactionRequired: noFiatTransactionRequired,
		},
	}
}

func newFlatFeeStandardLineForTest(servicePeriod timeutil.ClosedPeriod) *billing.StandardLine {
	chargeID := "charge-id"

	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				ID:        "line-id",
				Namespace: "namespace",
				CreatedAt: servicePeriod.From,
				UpdatedAt: servicePeriod.From,
				Name:      "flat fee",
			}),
			ManagedBy: billing.SystemManagedLine,
			Engine:    billing.LineEngineTypeChargeFlatFee,
			InvoiceID: "invoice-id",
			Currency:  currencyx.FiatCode("USD"),
			Period:    servicePeriod,
			InvoiceAt: servicePeriod.To,
			ChargeID:  &chargeID,
		},
		UsageBased: &billing.UsageBasedLine{},
	}
}
