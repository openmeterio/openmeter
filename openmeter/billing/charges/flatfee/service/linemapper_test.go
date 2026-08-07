package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestIsZeroFiatAmountOverageRunUsesConvertedOverage(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	charge := newFlatFeeCustomCurrencyCreditThenInvoiceChargeForTest(t, servicePeriod)

	tests := []struct {
		name                      string
		runTotals                 totals.Totals
		noFiatTransactionRequired bool
		expectZeroOverage         bool
	}{
		{
			name: "zero converted overage does not depend on transaction requirement",
			runTotals: totals.Totals{
				Amount:       alpacadecimal.NewFromInt(3),
				CreditsTotal: alpacadecimal.NewFromInt(3),
			},
			noFiatTransactionRequired: false,
			expectZeroOverage:         true,
		},
		{
			name: "positive converted overage is retained when no transaction is required",
			runTotals: totals.Totals{
				Amount: alpacadecimal.NewFromInt(3),
				Total:  alpacadecimal.NewFromInt(3),
			},
			noFiatTransactionRequired: true,
			expectZeroOverage:         false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given: a custom-currency run where converted overage and transaction requirement intentionally differ
			run := newFlatFeeCustomCurrencyRunForTest(
				servicePeriod,
				test.runTotals,
				test.noFiatTransactionRequired,
			)

			// when: zero-overage invoice handling is evaluated
			isZeroOverage := isZeroFiatAmountOverageRun(charge, run)

			// then: only the converted overage controls the result
			require.Equal(t, test.expectZeroOverage, isZeroOverage)
		})
	}
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
