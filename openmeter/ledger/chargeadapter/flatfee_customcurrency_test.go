package chargeadapter_test

import (
	"strings"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// TestOnFlatFeeCustomCurrencyOverageAccrued mirrors the usage-based boundary
// as a credit-purchase-equivalent flow: 100 ACME rated, 60 ACME covered by
// credits, 40 ACME uncovered overage is purchased at the persisted cost basis
// (0.25 USD/ACME) and immediately consumed by the same charge, then the
// resulting custom-currency receivable is converted once into a 10.00 USD
// fiat receivable. No spendable custom-currency balance or open custom
// receivable survives the accrual.
func TestOnFlatFeeCustomCurrencyOverageAccrued(t *testing.T) {
	env := newFlatFeeHandlerTestEnv(t)
	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	customCurrencyIdentity := customCurrencyValue.Reference()
	settlementCurrency := currencyx.Code("USD")
	costBasis := alpacadecimal.NewFromFloat(0.25)
	charge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, costBasis)
	run := env.newCustomOverageRun(totals.Totals{
		Amount:       alpacadecimal.NewFromInt(100),
		CreditsTotal: alpacadecimal.NewFromInt(60),
		Total:        alpacadecimal.NewFromInt(40),
	})

	result, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), flatfee.OnCustomCurrencyOverageAccruedInput{
		Charge: charge,
		Run:    run,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.TransactionGroup.TransactionGroupID)
	require.True(t, result.TotalFiatAmount.Equal(alpacadecimal.NewFromFloat(10)), "got %s", result.TotalFiatAmount)

	// The purchase is immediately and fully consumed by the same charge: no
	// spendable custom-currency FBO balance and no open custom receivable survive.
	fboSubAccount := env.FBOSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)
	require.Equal(t, 0.0, env.sumBalance(t, fboSubAccount).InexactFloat64())
	require.Equal(t, 0.0, env.sumBalance(t, env.ReceivableSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)).InexactFloat64())

	// The consumed amount is accrued natively, preserving cost basis and fiat provenance.
	accruedSubAccount := env.AccruedSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, &costBasis, lo.ToPtr(testChargeTaxCodeID))
	require.Equal(t, 40.0, env.sumBalance(t, accruedSubAccount).InexactFloat64())

	// The already-agreed, rounded fiat amount becomes the open invoice receivable.
	require.Equal(t, -10.0, env.sumBalance(t, env.ReceivableSubAccountWithCostBasis(t, &costBasis)).InexactFloat64())

	// Brokerage carries the exact fiat and native legs of the conversion.
	require.Equal(t, 10.0, env.sumBalance(t, env.BrokerageSubAccountForCurrency(t, currencies.NewCurrencyReference(settlementCurrency), nil, costBasis)).InexactFloat64())
	require.Equal(t, -40.0, env.sumBalance(t, env.BrokerageSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)).InexactFloat64())

	// The atomic group uses the same economic steps as a paid credit purchase,
	// with the purchased credit consumed immediately before its receivable is
	// converted into the invoice currency.
	templateCodes := make([]string, 0, 3)
	for _, txAnnotations := range env.transactionAnnotations(t, result.TransactionGroup.TransactionGroupID) {
		require.Equal(t, charge.ID, txAnnotations[ledger.AnnotationChargeID], "transaction missing charge annotation: %v", txAnnotations)
		templateCode, err := ledger.TransactionTemplateCodeFromAnnotations(txAnnotations)
		require.NoError(t, err)
		templateCodes = append(templateCodes, templateCode)
	}
	require.ElementsMatch(t, []string{
		transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}),
		transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}),
		transactions.TemplateCode(transactions.ConvertCurrencyTemplate{}),
	}, templateCodes)

	// Like a real credit purchase, issuance identifies only the credit source.
	// Consumption adds the spend identity while preserving that source.
	entries := env.TransactionGroupEntries(t, result.TransactionGroup.TransactionGroupID)
	fboEntries := env.EntriesForSubAccount(entries, fboSubAccount)
	require.Len(t, fboEntries, 2, "issue and immediate consumption must each post to the custom FBO route")
	issueEntry := fboEntries[0]
	consumeEntry := fboEntries[1]
	if issueEntry.Amount.IsNegative() {
		issueEntry, consumeEntry = consumeEntry, issueEntry
	}
	require.True(t, issueEntry.Amount.IsPositive())
	require.NotNil(t, issueEntry.SourceChargeID)
	require.Equal(t, charge.ID, strings.TrimSpace(*issueEntry.SourceChargeID))
	require.Nil(t, issueEntry.SpendChargeID)
	require.True(t, consumeEntry.Amount.IsNegative())
	require.NotNil(t, consumeEntry.SourceChargeID)
	require.Equal(t, charge.ID, strings.TrimSpace(*consumeEntry.SourceChargeID))
	require.NotNil(t, consumeEntry.SpendChargeID)
	require.Equal(t, charge.ID, strings.TrimSpace(*consumeEntry.SpendChargeID))

	accruedEntries := env.EntriesForSubAccount(entries, accruedSubAccount)
	require.Len(t, accruedEntries, 1)
	require.True(t, accruedEntries[0].Amount.Equal(alpacadecimal.NewFromInt(40)))
	require.NotNil(t, accruedEntries[0].SourceChargeID)
	require.Equal(t, charge.ID, strings.TrimSpace(*accruedEntries[0].SourceChargeID))
	require.NotNil(t, accruedEntries[0].SpendChargeID)
	require.Equal(t, charge.ID, strings.TrimSpace(*accruedEntries[0].SpendChargeID))
}

// TestOnFlatFeeCustomCurrencyOverageAccrued_RejectsNonPositiveFiatOutcomes
// mirrors the usage-based guarantee that neither a zero native overage nor an
// overage that rounds to zero fiat ever produces a ledger transaction.
func TestOnFlatFeeCustomCurrencyOverageAccrued_RejectsNonPositiveFiatOutcomes(t *testing.T) {
	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)

	t.Run("zero native overage", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)
		charge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, alpacadecimal.NewFromFloat(0.25))

		result, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), flatfee.OnCustomCurrencyOverageAccruedInput{
			Charge: charge,
			Run:    env.newCustomOverageRun(totals.Totals{Amount: alpacadecimal.Zero, Total: alpacadecimal.Zero}),
		})
		require.Error(t, err)
		require.Empty(t, result.TransactionGroup.TransactionGroupID)
	})

	t.Run("native overage rounds to zero fiat", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)
		// 1 ACME at a 0.001 USD/ACME rate rounds to 0.00 USD at 2-decimal fiat
		// precision: the native amount is positive, but there is nothing to book.
		charge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, alpacadecimal.NewFromFloat(0.001))

		result, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), flatfee.OnCustomCurrencyOverageAccruedInput{
			Charge: charge,
			Run:    env.newCustomOverageRun(totals.Totals{Amount: alpacadecimal.NewFromInt(1), Total: alpacadecimal.NewFromInt(1)}),
		})
		require.Error(t, err)
		require.Empty(t, result.TransactionGroup.TransactionGroupID)
	})
}

// TestOnFlatFeeCustomCurrencyOverageAccrued_UsesPersistedCostBasis proves the
// converted fiat amount is a pure function of the charge's persisted
// ResolvedCostBasis snapshot, mirroring the usage-based guarantee.
func TestOnFlatFeeCustomCurrencyOverageAccrued_UsesPersistedCostBasis(t *testing.T) {
	env := newFlatFeeHandlerTestEnv(t)
	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	overage := totals.Totals{
		Amount:       alpacadecimal.NewFromInt(100),
		CreditsTotal: alpacadecimal.NewFromInt(60),
		Total:        alpacadecimal.NewFromInt(40),
	}

	historicalCharge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, alpacadecimal.NewFromFloat(0.25))
	historicalResult, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), flatfee.OnCustomCurrencyOverageAccruedInput{
		Charge: historicalCharge,
		Run:    env.newCustomOverageRun(overage),
	})
	require.NoError(t, err)
	require.True(t, historicalResult.TotalFiatAmount.Equal(alpacadecimal.NewFromFloat(10)), "got %s", historicalResult.TotalFiatAmount)

	// A later cost-basis change must not affect the amount computed from the
	// already-persisted (historical) rate above.
	laterRateCharge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, alpacadecimal.NewFromFloat(0.5))
	laterResult, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), flatfee.OnCustomCurrencyOverageAccruedInput{
		Charge: laterRateCharge,
		Run:    env.newCustomOverageRun(overage),
	})
	require.NoError(t, err)
	require.True(t, laterResult.TotalFiatAmount.Equal(alpacadecimal.NewFromFloat(20)), "got %s", laterResult.TotalFiatAmount)
	require.False(t, historicalResult.TotalFiatAmount.Equal(laterResult.TotalFiatAmount))
}

// TestOnFlatFeeCustomCurrencyPaymentLifecycle exercises accrual, authorization,
// and settlement for the converted fiat overage, verifying the whole payment
// lifecycle stays in the invoice fiat currency for a custom-currency charge,
// using the charge's persisted cost basis rather than a fixed cost basis of 1.
func TestOnFlatFeeCustomCurrencyPaymentLifecycle(t *testing.T) {
	env := newFlatFeeHandlerTestEnv(t)
	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	customCurrencyIdentity := customCurrencyValue.Reference()
	settlementCurrency := currencyx.Code("USD")
	costBasis := alpacadecimal.NewFromFloat(0.25)
	charge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, costBasis)
	run := env.newCustomOverageRun(totals.Totals{
		Amount:       alpacadecimal.NewFromInt(100),
		CreditsTotal: alpacadecimal.NewFromInt(60),
		Total:        alpacadecimal.NewFromInt(40),
	})

	accrualResult, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), flatfee.OnCustomCurrencyOverageAccruedInput{
		Charge: charge,
		Run:    run,
	})
	require.NoError(t, err)

	fiatOverage := accrualResult.TotalFiatAmount

	authorizeRef, err := env.handler.OnPaymentAuthorized(t.Context(), flatfee.OnPaymentAuthorizedInput{
		Charge:     charge,
		Run:        run,
		EventAt:    env.Now(),
		FiatAmount: fiatOverage,
	})
	require.NoError(t, err)
	require.NotEmpty(t, authorizeRef.TransactionGroupID)
	for _, entry := range env.TransactionGroupEntries(t, authorizeRef.TransactionGroupID) {
		require.NotNil(t, entry.SourceChargeID)
		require.Equal(t, charge.ID, strings.TrimSpace(*entry.SourceChargeID))
		require.Nil(t, entry.SpendChargeID)
	}

	require.Equal(t, 0.0, env.sumBalance(t, env.ReceivableSubAccountWithCostBasis(t, &costBasis)).InexactFloat64())
	require.Equal(t, -10.0, env.sumBalance(t, env.ReceivableSubAccountWithCostBasisAndStatus(t, &costBasis, ledger.TransactionAuthorizationStatusAuthorized)).InexactFloat64())

	settleRef, err := env.handler.OnPaymentSettled(t.Context(), flatfee.OnPaymentSettledInput{
		Charge:     charge,
		Run:        run,
		EventAt:    env.Now(),
		FiatAmount: fiatOverage,
	})
	require.NoError(t, err)
	require.NotEmpty(t, settleRef.TransactionGroupID)
	for _, entry := range env.TransactionGroupEntries(t, settleRef.TransactionGroupID) {
		require.NotNil(t, entry.SourceChargeID)
		require.Equal(t, charge.ID, strings.TrimSpace(*entry.SourceChargeID))
		require.Nil(t, entry.SpendChargeID)
	}

	require.Equal(t, 0.0, env.sumBalance(t, env.ReceivableSubAccountWithCostBasisAndStatus(t, &costBasis, ledger.TransactionAuthorizationStatusAuthorized)).InexactFloat64())
	require.Equal(t, -10.0, env.sumBalance(t, env.WashSubAccountWithCostBasis(t, &costBasis)).InexactFloat64())

	// Authorization and settlement never touch the custom-currency side: FBO
	// stays empty, the custom receivable stays closed, and the purchased
	// credit that was consumed at accrual time remains the only trace.
	require.Equal(t, 0.0, env.sumBalance(t, env.FBOSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)).InexactFloat64())
	require.Equal(t, 0.0, env.sumBalance(t, env.ReceivableSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)).InexactFloat64())
	require.Equal(t, 40.0, env.sumBalance(t, env.AccruedSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, &costBasis, lo.ToPtr(testChargeTaxCodeID))).InexactFloat64())
}

// newCustomCurrencyCreditThenInvoiceCharge builds a credit_then_invoice charge
// with a manually resolved and persisted cost basis, the shape a real charge
// has once its dynamic cost basis has been resolved and snapshotted.
func (e *flatFeeHandlerTestEnv) newCustomCurrencyCreditThenInvoiceCharge(t *testing.T, customCurrencyValue currencies.Currency, costBasis alpacadecimal.Decimal) flatfee.Charge {
	t.Helper()

	usd, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	now := time.Now().UTC()
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-time.Hour),
		To:   now,
	}
	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: usd,
		Rate:         costBasis,
	})

	return flatfee.Charge{
		ChargeBase: flatfee.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: e.Namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				ID: "flat-fee-charge-cc-cti",
			},
			Intent: flatfee.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.SystemManagedLine,
					CustomerID: e.CustomerID.ID,
					Currency:   customCurrencyValue,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: testChargeTaxCodeID,
					},
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "Flat fee (custom currency, credit then invoice)",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:             now,
					PaymentTerm:           productcatalog.InAdvancePaymentTerm,
					ProRating:             productcatalog.ProRatingConfig{},
					AmountBeforeProration: alpacadecimal.NewFromInt(100),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				CostBasis:      &costBasisIntent,
			}.AsOverridableIntent(),
			Status: flatfee.StatusActive,
			State: flatfee.State{
				AmountAfterProration: alpacadecimal.NewFromInt(100),
				CostBasisID:          lo.ToPtr("cost-basis-1"),
				ResolvedCostBasis: &costbasis.State{
					CostBasis:  costBasis,
					ResolvedAt: now,
				},
			},
		},
	}
}

func (e *flatFeeHandlerTestEnv) newCustomOverageRun(overageTotals totals.Totals) flatfee.RealizationRun {
	now := time.Now().UTC()

	return flatfee.RealizationRun{
		RealizationRunBase: flatfee.RealizationRunBase{
			ID: flatfee.RealizationRunID(models.NamespacedID{
				Namespace: e.Namespace,
				ID:        "run-1",
			}),
			ManagedModel: models.ManagedModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			LineID:      lo.ToPtr("line-1"),
			Type:        flatfee.RealizationRunTypeFinalRealization,
			InitialType: flatfee.RealizationRunTypeFinalRealization,
			ServicePeriod: timeutil.ClosedPeriod{
				From: now.Add(-time.Hour),
				To:   now,
			},
			AmountAfterProration: overageTotals.Amount,
			Totals:               overageTotals,
		},
	}
}
