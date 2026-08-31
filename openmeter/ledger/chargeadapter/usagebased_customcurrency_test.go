package chargeadapter_test

import (
	"strings"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	chargeusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
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

// TestOnUsageBasedCustomCurrencyOverageAccrued exercises the credit_then_invoice
// boundary as a credit-purchase-equivalent ledger flow: 100 ACME rated, 60 ACME
// covered by credits, and 40 ACME uncovered overage is issued at the persisted
// cost basis (0.25 USD/ACME) and immediately consumed by the same charge. The
// resulting custom-currency receivable is converted once into a 10.00 USD fiat
// receivable. No spendable custom-currency balance or open custom receivable
// survives the accrual.
func TestOnUsageBasedCustomCurrencyOverageAccrued(t *testing.T) {
	env := newUsageBasedHandlerTestEnv(t)
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

	result, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), chargeusagebased.OnCustomCurrencyOverageAccruedInput{
		Charge: charge,
		Run:    run,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.TransactionGroup.TransactionGroupID)
	require.True(t, result.TotalFiatAmount.Equal(alpacadecimal.NewFromFloat(10)), "got %s", result.TotalFiatAmount)

	// The purchase is immediately and fully consumed by the same charge: no
	// spendable custom-currency FBO balance and no open custom receivable survive.
	fboSubAccount := env.FBOSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)
	customReceivableSubAccount := env.ReceivableSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)
	require.True(t, env.sumBalance(t, fboSubAccount).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, customReceivableSubAccount).Equal(alpacadecimal.Zero))

	// The consumed amount is accrued natively, preserving cost basis and fiat provenance.
	accruedSubAccount := env.AccruedSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, &costBasis, lo.ToPtr(testChargeTaxCodeID))
	require.True(t, env.sumBalance(t, accruedSubAccount).Equal(alpacadecimal.NewFromInt(40)))

	// The already-agreed, rounded fiat amount becomes the open invoice receivable.
	require.True(t, env.sumBalance(t, env.ReceivableSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(-10)))

	// Brokerage carries the exact fiat and native legs of the conversion.
	require.True(t, env.sumBalance(t, env.BrokerageSubAccountForCurrency(t, currencies.NewCurrencyReference(settlementCurrency), nil, costBasis)).Equal(alpacadecimal.NewFromInt(10)))
	require.True(t, env.sumBalance(t, env.BrokerageSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)).Equal(alpacadecimal.NewFromInt(-40)))

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

	// Issuance and FX close the same source-attributed custom receivable bucket.
	customReceivableEntries := env.EntriesForSubAccount(entries, customReceivableSubAccount)
	require.Len(t, customReceivableEntries, 2)
	for _, entry := range customReceivableEntries {
		require.NotNil(t, entry.SourceChargeID)
		require.Equal(t, charge.ID, strings.TrimSpace(*entry.SourceChargeID))
		require.Nil(t, entry.SpendChargeID)
	}
}

func TestOnUsageBasedCustomCurrencyOverageAccruedCorrection(t *testing.T) {
	env := newUsageBasedHandlerTestEnv(t)
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

	// given: invoice finalization has persisted the gross overage booking
	result, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), chargeusagebased.OnCustomCurrencyOverageAccruedInput{
		Charge: charge,
		Run:    run,
	})
	require.NoError(t, err)
	run.InvoiceUsage = &invoicedusage.AccruedUsage{
		ServicePeriod:     charge.Intent.GetEffectiveServicePeriod(),
		Totals:            totals.Totals{Amount: result.TotalFiatAmount, Total: result.TotalFiatAmount},
		LedgerTransaction: &result.TransactionGroup,
	}

	// when: failed invoice synchronization deletes the prepared run
	err = env.handler.OnCustomCurrencyOverageAccruedCorrection(t.Context(), chargeusagebased.OnCustomCurrencyOverageAccruedCorrectionInput{
		Charge: charge,
		Run:    run,
	})
	require.NoError(t, err)

	// then: every native and fiat leg is fully reversed
	fboSubAccount := env.FBOSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)
	customReceivableSubAccount := env.ReceivableSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)
	accruedSubAccount := env.AccruedSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, &costBasis, lo.ToPtr(testChargeTaxCodeID))
	require.True(t, env.sumBalance(t, fboSubAccount).IsZero())
	require.True(t, env.sumBalance(t, customReceivableSubAccount).IsZero())
	require.True(t, env.sumBalance(t, accruedSubAccount).IsZero())
	require.True(t, env.sumBalance(t, env.ReceivableSubAccountWithCostBasis(t, &costBasis)).IsZero())
	require.True(t, env.sumBalance(t, env.BrokerageSubAccountForCurrency(t, currencies.NewCurrencyReference(settlementCurrency), nil, costBasis)).IsZero())
	require.True(t, env.sumBalance(t, env.BrokerageSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)).IsZero())

	corrections, err := env.Deps.HistoricalLedger.ListTransactions(t.Context(), ledger.ListTransactionsInput{
		Namespace: env.Namespace,
		Limit:     10,
		AnnotationFilters: map[string]string{
			ledger.AnnotationChargeID:             charge.ID,
			ledger.AnnotationTransactionDirection: string(ledger.TransactionDirectionCorrection),
		},
	})
	require.NoError(t, err)
	require.Len(t, corrections.Items, 3)

	templateCodes := make([]string, 0, len(corrections.Items))
	for _, transaction := range corrections.Items {
		require.True(t, run.ServicePeriodTo.Equal(transaction.BookedAt()), "expected %s, got %s", run.ServicePeriodTo, transaction.BookedAt())
		templateCode, err := ledger.TransactionTemplateCodeFromAnnotations(transaction.Annotations())
		require.NoError(t, err)
		templateCodes = append(templateCodes, templateCode)
	}
	require.ElementsMatch(t, []string{
		transactions.TemplateCode(transactions.ConvertCurrencyTemplate{}),
		transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}),
		transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}),
	}, templateCodes)
}

func TestOnUsageBasedCustomCurrencyOverageAccruedCorrection_NoLedgerTransaction(t *testing.T) {
	env := newUsageBasedHandlerTestEnv(t)
	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	charge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, alpacadecimal.NewFromFloat(0.001))
	run := env.newCustomOverageRun(totals.Totals{
		Amount: alpacadecimal.NewFromInt(1),
		Total:  alpacadecimal.NewFromInt(1),
	})
	run.InvoiceUsage = &invoicedusage.AccruedUsage{
		ServicePeriod: charge.Intent.GetEffectiveServicePeriod(),
		Totals:        totals.Totals{},
	}

	// A positive native overage that rounded to zero fiat persisted no ledger
	// reference, so cleanup has no gross booking to reverse.
	err := env.handler.OnCustomCurrencyOverageAccruedCorrection(t.Context(), chargeusagebased.OnCustomCurrencyOverageAccruedCorrectionInput{
		Charge: charge,
		Run:    run,
	})
	require.NoError(t, err)
}

func TestOnUsageBasedCustomCurrencyOverageUsesFiatCreditsToCoverReceivable(t *testing.T) {
	env := newUsageBasedHandlerTestEnv(t)
	sourceChargeID := "fiat-credit-source"
	fbo := env.fundPriorityForSource(t, 1, 6, sourceChargeID)

	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	costBasis := alpacadecimal.NewFromFloat(0.25)
	charge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, costBasis)
	run := env.newCustomOverageRun(totals.Totals{
		Amount: alpacadecimal.NewFromInt(40),
		Total:  alpacadecimal.NewFromInt(40),
	})

	// given: the full custom overage has already created a 10 USD receivable
	_, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), chargeusagebased.OnCustomCurrencyOverageAccruedInput{
		Charge: charge,
		Run:    run,
	})
	require.NoError(t, err)

	// when: charges requests 6 USD of existing fiat credit to settle it
	allocations, err := env.handler.OnAllocateFiatOverageCredits(t.Context(), chargeusagebased.AllocateFiatOverageCreditsInput{
		Charge:           charge,
		Run:              run,
		BookedAt:         env.Now(),
		AmountToAllocate: alpacadecimal.NewFromInt(6),
	})
	require.NoError(t, err)
	require.Len(t, allocations, 1)
	require.Equal(t, float64(6), allocations[0].Amount.InexactFloat64())

	// then: the gross FX-route receivable remains intact, while the source
	// credit's own route offsets it at the receivable-account level.
	zeroCostBasis := alpacadecimal.Zero
	require.Equal(t, float64(0), env.sumBalance(t, fbo).InexactFloat64())
	require.Equal(t, float64(6), env.sumBalance(t, env.ReceivableSubAccountWithCostBasis(t, &zeroCostBasis)).InexactFloat64())
	require.Equal(t, float64(-10), env.sumBalance(t, env.ReceivableSubAccountWithCostBasis(t, &costBasis)).InexactFloat64())

	entries := env.TransactionGroupEntries(t, allocations[0].LedgerTransaction.TransactionGroupID)
	for _, entry := range entries {
		require.NotNil(t, entry.SourceChargeID)
		require.Equal(t, sourceChargeID, strings.TrimSpace(*entry.SourceChargeID))
		require.NotNil(t, entry.SpendChargeID)
		require.Equal(t, charge.ID, strings.TrimSpace(*entry.SpendChargeID))
	}

	// and: correcting the billing allocation restores the same FBO source and
	// removes its receivable offset.
	fiatCurrency, err := charge.Intent.GetCostBasisIntent().GetFiatCurrency()
	require.NoError(t, err)
	realizations := env.realizationsFromAllocations(allocations)
	request, err := realizations.CreateCorrectionRequest(alpacadecimal.NewFromInt(-6), fiatCurrency)
	require.NoError(t, err)
	run.FiatOverageCreditRealizations = realizations

	corrections, err := env.handler.OnCorrectFiatOverageCreditAllocations(t.Context(), chargeusagebased.CorrectFiatOverageCreditAllocationsInput{
		Charge:      charge,
		Run:         run,
		BookedAt:    env.Now(),
		Corrections: request,
	})
	require.NoError(t, err)
	require.Len(t, corrections, 1)
	require.Equal(t, float64(6), env.sumBalance(t, fbo).InexactFloat64())
	require.Equal(t, float64(0), env.sumBalance(t, env.ReceivableSubAccountWithCostBasis(t, &zeroCostBasis)).InexactFloat64())
	require.Equal(t, float64(-10), env.sumBalance(t, env.ReceivableSubAccountWithCostBasis(t, &costBasis)).InexactFloat64())
}

// TestOnUsageBasedCustomCurrencyOverageAccrued_RejectsNonPositiveFiatOutcomes
// proves neither a zero native overage nor an overage that rounds to zero
// fiat ever produces a ledger transaction: both fail before any transaction
// group is committed, matching the pre-existing no-op contract the run-level
// caller (BookAccruedInvoiceUsage) already relies on when it skips calling
// this handler for a zero invoice line total.
func TestOnUsageBasedCustomCurrencyOverageAccrued_RejectsNonPositiveFiatOutcomes(t *testing.T) {
	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)

	t.Run("zero native overage", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)
		charge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, alpacadecimal.NewFromFloat(0.25))

		result, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), chargeusagebased.OnCustomCurrencyOverageAccruedInput{
			Charge: charge,
			Run: env.newCustomOverageRun(totals.Totals{
				Amount:       alpacadecimal.NewFromInt(60),
				CreditsTotal: alpacadecimal.NewFromInt(60),
				Total:        alpacadecimal.Zero,
			}),
		})
		require.Error(t, err)
		require.Empty(t, result.TransactionGroup.TransactionGroupID)
	})

	t.Run("native overage rounds to zero fiat", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)
		// 1 ACME at a 0.001 USD/ACME rate rounds to 0.00 USD at 2-decimal fiat
		// precision: the native amount is positive, but there is nothing to book.
		charge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, alpacadecimal.NewFromFloat(0.001))

		result, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), chargeusagebased.OnCustomCurrencyOverageAccruedInput{
			Charge: charge,
			Run: env.newCustomOverageRun(totals.Totals{
				Amount: alpacadecimal.NewFromInt(1),
				Total:  alpacadecimal.NewFromInt(1),
			}),
		})
		require.Error(t, err)
		require.Empty(t, result.TransactionGroup.TransactionGroupID)
	})
}

// TestOnUsageBasedCustomCurrencyOverageAccrued_UsesPersistedCostBasis proves the
// converted fiat amount is a pure function of the charge's persisted
// ResolvedCostBasis snapshot: two charges with the same custom-currency
// overage but different persisted rates produce different fiat amounts, and
// neither call re-resolves a "current" rate.
func TestOnUsageBasedCustomCurrencyOverageAccrued_UsesPersistedCostBasis(t *testing.T) {
	env := newUsageBasedHandlerTestEnv(t)
	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	overage := totals.Totals{
		Amount:       alpacadecimal.NewFromInt(100),
		CreditsTotal: alpacadecimal.NewFromInt(60),
		Total:        alpacadecimal.NewFromInt(40),
	}

	historicalCharge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, alpacadecimal.NewFromFloat(0.25))
	historicalResult, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), chargeusagebased.OnCustomCurrencyOverageAccruedInput{
		Charge: historicalCharge,
		Run:    env.newCustomOverageRun(overage),
	})
	require.NoError(t, err)
	require.True(t, historicalResult.TotalFiatAmount.Equal(alpacadecimal.NewFromFloat(10)), "got %s", historicalResult.TotalFiatAmount)

	// A later cost-basis change must not affect the amount computed from the
	// already-persisted (historical) rate above.
	laterRateCharge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, alpacadecimal.NewFromFloat(0.5))
	laterResult, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), chargeusagebased.OnCustomCurrencyOverageAccruedInput{
		Charge: laterRateCharge,
		Run:    env.newCustomOverageRun(overage),
	})
	require.NoError(t, err)
	require.True(t, laterResult.TotalFiatAmount.Equal(alpacadecimal.NewFromFloat(20)), "got %s", laterResult.TotalFiatAmount)
	require.False(t, historicalResult.TotalFiatAmount.Equal(laterResult.TotalFiatAmount))
}

// TestOnUsageBasedCustomCurrencyOverageAccrued_ProgressiveRuns proves that two
// sequential overage runs against the same persisted cost basis (progressive
// billing) each issue, consume, and convert only their own newly uncovered
// increment without double booking. Each increment is converted independently;
// this exact-rate test makes no claim about cross-run rounding, for which
// charges retain no remainder state. A later run against the same charge and
// persisted cost basis lands on the exact same purchase route.
func TestOnUsageBasedCustomCurrencyOverageAccrued_ProgressiveRuns(t *testing.T) {
	env := newUsageBasedHandlerTestEnv(t)
	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	customCurrencyIdentity := customCurrencyValue.Reference()
	settlementCurrency := currencyx.Code("USD")
	costBasis := alpacadecimal.NewFromFloat(0.25)
	charge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, costBasis)

	firstRun := env.newCustomOverageRun(totals.Totals{
		Amount:       alpacadecimal.NewFromInt(100),
		CreditsTotal: alpacadecimal.NewFromInt(60),
		Total:        alpacadecimal.NewFromInt(40),
	})
	firstResult, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), chargeusagebased.OnCustomCurrencyOverageAccruedInput{
		Charge: charge,
		Run:    firstRun,
	})
	require.NoError(t, err)
	require.True(t, firstResult.TotalFiatAmount.Equal(alpacadecimal.NewFromFloat(10)), "got %s", firstResult.TotalFiatAmount)

	// Second run represents only the incremental delta accrued afterwards.
	secondRun := env.newCustomOverageRun(totals.Totals{
		Amount:       alpacadecimal.NewFromInt(20),
		CreditsTotal: alpacadecimal.Zero,
		Total:        alpacadecimal.NewFromInt(20),
	})
	secondResult, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), chargeusagebased.OnCustomCurrencyOverageAccruedInput{
		Charge: charge,
		Run:    secondRun,
	})
	require.NoError(t, err)
	require.True(t, secondResult.TotalFiatAmount.Equal(alpacadecimal.NewFromFloat(5)), "got %s", secondResult.TotalFiatAmount)
	require.NotEqual(t, firstResult.TransactionGroup.TransactionGroupID, secondResult.TransactionGroup.TransactionGroupID)

	// Both runs purchase-and-consume on the same route: FBO and the open custom
	// receivable stay at zero cumulatively, never accumulating spendable credit.
	require.True(t, env.sumBalance(t, env.FBOSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.ReceivableSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.AccruedSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, &costBasis, lo.ToPtr(testChargeTaxCodeID))).Equal(alpacadecimal.NewFromInt(60)))

	require.True(t, env.sumBalance(t, env.ReceivableSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(-15)))
	require.True(t, env.sumBalance(t, env.BrokerageSubAccountForCurrency(t, currencies.NewCurrencyReference(settlementCurrency), nil, costBasis)).Equal(alpacadecimal.NewFromInt(15)))
	require.True(t, env.sumBalance(t, env.BrokerageSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)).Equal(alpacadecimal.NewFromInt(-60)))
}

// TestOnUsageBasedCustomCurrencyPaymentLifecycle exercises accrual, authorization,
// and settlement for the converted fiat overage, verifying the whole payment
// lifecycle stays in the invoice fiat currency for a custom-currency charge,
// using the charge's persisted cost basis rather than a fixed cost basis of 1.
func TestOnUsageBasedCustomCurrencyPaymentLifecycle(t *testing.T) {
	env := newUsageBasedHandlerTestEnv(t)
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

	accrualResult, err := env.handler.OnCustomCurrencyOverageAccrued(t.Context(), chargeusagebased.OnCustomCurrencyOverageAccruedInput{
		Charge: charge,
		Run:    run,
	})
	require.NoError(t, err)

	fiatOverage := accrualResult.TotalFiatAmount
	run.LineID = lo.ToPtr("line-1")

	authorizeRef, err := env.handler.OnPaymentAuthorized(t.Context(), chargeusagebased.OnPaymentAuthorizedInput{
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

	require.True(t, env.sumBalance(t, env.ReceivableSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.ReceivableSubAccountWithCostBasisAndStatus(t, &costBasis, ledger.TransactionAuthorizationStatusAuthorized)).Equal(alpacadecimal.NewFromInt(-10)))

	settleRef, err := env.handler.OnPaymentSettled(t.Context(), chargeusagebased.OnPaymentSettledInput{
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

	require.True(t, env.sumBalance(t, env.ReceivableSubAccountWithCostBasisAndStatus(t, &costBasis, ledger.TransactionAuthorizationStatusAuthorized)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.WashSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(-10)))

	// Authorization and settlement never touch the custom-currency side: FBO
	// stays empty, the custom receivable stays closed, and the purchased
	// credit that was consumed at accrual time remains the only trace.
	require.True(t, env.sumBalance(t, env.FBOSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.ReceivableSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.AccruedSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, &costBasis, lo.ToPtr(testChargeTaxCodeID))).Equal(alpacadecimal.NewFromInt(40)))

	// No custom-currency receivable or payment exposure is ever created for the overage.
	require.True(t, env.sumBalance(t, env.customReceivableSubAccountForUsageBasedFeature(t, customCurrencyValue.GetCode(), customCurrencyValue.Reference(), "api_requests")).Equal(alpacadecimal.Zero))
}

// TestOnUsageBasedCreditsOnlyUsageAccrued_CustomCurrency exercises credit_only
// usage collection in a custom currency through the real collector: first
// fully covered by funded custom FBO, then an uncovered amount that must
// advance a custom-currency receivable exactly like the fiat path does.
func TestOnUsageBasedCreditsOnlyUsageAccrued_CustomCurrency(t *testing.T) {
	settlementCurrency := currencyx.Code("USD")
	costBasis := alpacadecimal.NewFromFloat(0.25)

	t.Run("collects from funded custom FBO preserving cost basis and fiat source", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)
		customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
		customCurrency := customCurrencyValue.GetCode()
		customCurrencyIdentity := customCurrencyValue.Reference()
		charge := env.newCustomCurrencyCreditsOnlyCharge(t, customCurrencyValue)

		env.fundCustomFBO(t, customCurrency, customCurrencyIdentity, &settlementCurrency, costBasis, alpacadecimal.NewFromInt(30))

		realizations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedInput{
			Charge:           charge,
			Run:              env.newRun(),
			BookedAt:         env.Now(),
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.NoError(t, err)
		require.Len(t, realizations, 1)
		require.True(t, realizations[0].Amount.Equal(alpacadecimal.NewFromInt(30)))

		require.True(t, env.sumBalance(t, env.FBOSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.AccruedSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, &costBasis, lo.ToPtr(testChargeTaxCodeID))).Equal(alpacadecimal.NewFromInt(30)))
	})

	t.Run("credit_only advances an uncovered custom currency amount", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)
		customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
		customCurrency := customCurrencyValue.GetCode()
		customCurrencyIdentity := customCurrencyValue.Reference()
		charge := env.newCustomCurrencyCreditsOnlyCharge(t, customCurrencyValue)

		realizations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedInput{
			Charge:           charge,
			Run:              env.newRun(),
			BookedAt:         env.Now(),
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.NoError(t, err)
		require.Len(t, realizations, 1)
		require.True(t, realizations[0].Amount.Equal(alpacadecimal.NewFromInt(30)))

		require.True(t, env.sumBalance(t, env.customReceivableSubAccountForUsageBasedFeature(t, customCurrency, customCurrencyIdentity, "api_requests")).Equal(alpacadecimal.NewFromInt(-30)))
		require.True(t, env.sumBalance(t, env.customUnknownAccruedSubAccountForUsageBased(t, customCurrency, customCurrencyIdentity)).Equal(alpacadecimal.NewFromInt(30)))
	})
}

// TestOnUsageBasedCreditsOnlyUsageAccruedCorrection_CustomCurrency exercises
// correcting a credit_only custom-currency accrual that advanced (unknown
// cost basis) because FBO couldn't cover it. Unlike the credit-purchase
// lifecycle, credit_only correction never leaves the custom currency (no FX
// conversion involved), so a full reversal must zero out the receivable,
// FBO, and accrued buckets exactly like the fiat "reverses advance-backed
// accrual" case does.
func TestOnUsageBasedCreditsOnlyUsageAccruedCorrection_CustomCurrency(t *testing.T) {
	env := newUsageBasedHandlerTestEnv(t)
	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	customCurrency := customCurrencyValue.GetCode()
	customCurrencyIdentity := customCurrencyValue.Reference()
	charge := env.newCustomCurrencyCreditsOnlyCharge(t, customCurrencyValue)

	run := env.newRun()
	allocations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedInput{
		Charge:           charge,
		Run:              run,
		BookedAt:         env.Now(),
		AmountToAllocate: alpacadecimal.NewFromInt(30),
	})
	require.NoError(t, err)
	require.Len(t, allocations, 1)

	run.CreditsAllocated = env.realizationsFromAllocations(allocations)

	correctionsRequest, err := run.CreditsAllocated.CreateCorrectionRequest(alpacadecimal.NewFromInt(-30), customCurrencyValue)
	require.NoError(t, err)

	corrections, err := env.handler.OnCreditsOnlyUsageAccruedCorrection(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedCorrectionInput{
		Charge:      charge,
		Run:         run,
		BookedAt:    env.Now(),
		Corrections: correctionsRequest,
	})
	require.NoError(t, err)
	require.Len(t, corrections, 1)
	require.True(t, corrections[0].Amount.Equal(alpacadecimal.NewFromInt(-30)))

	require.True(t, env.sumBalance(t, env.customReceivableSubAccountForUsageBasedFeature(t, customCurrency, customCurrencyIdentity, "api_requests")).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.customUnknownFboSubAccountForUsageBased(t, customCurrency, customCurrencyIdentity)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.customUnknownAccruedSubAccountForUsageBased(t, customCurrency, customCurrencyIdentity)).Equal(alpacadecimal.Zero))
}

// TestOnUsageBasedCreditThenInvoiceCreditAllocationCorrection_CustomCurrency
// proves that correcting a credit_then_invoice charge's native credit
// allocation stays entirely in the custom currency: the correction never
// touches the fiat overage side (accrual, receivable, or payment routes),
// matching the domain rule that only the uncovered overage - never the
// credit allocation itself - crosses the fiat boundary.
func TestOnUsageBasedCreditThenInvoiceCreditAllocationCorrection_CustomCurrency(t *testing.T) {
	env := newUsageBasedHandlerTestEnv(t)
	settlementCurrency := currencyx.Code("USD")
	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	customCurrency := customCurrencyValue.GetCode()
	customCurrencyIdentity := customCurrencyValue.Reference()
	costBasis := alpacadecimal.NewFromFloat(0.25)
	charge := env.newCustomCurrencyCreditThenInvoiceCharge(t, customCurrencyValue, costBasis)

	// credit_then_invoice only allocates from already-funded credit; unlike
	// credit_only it never advances an uncovered amount into an unknown bucket.
	env.fundCustomFBO(t, customCurrency, customCurrencyIdentity, &settlementCurrency, costBasis, alpacadecimal.NewFromInt(60))

	run := env.newRun()
	allocations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedInput{
		Charge:           charge,
		Run:              run,
		BookedAt:         env.Now(),
		AmountToAllocate: alpacadecimal.NewFromInt(60),
	})
	require.NoError(t, err)
	require.Len(t, allocations, 1)
	require.True(t, allocations[0].Amount.Equal(alpacadecimal.NewFromInt(60)))

	run.CreditsAllocated = env.realizationsFromAllocations(allocations)

	correctionsRequest, err := run.CreditsAllocated.CreateCorrectionRequest(alpacadecimal.NewFromInt(-20), customCurrencyValue)
	require.NoError(t, err)

	corrections, err := env.handler.OnCreditsOnlyUsageAccruedCorrection(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedCorrectionInput{
		Charge:      charge,
		Run:         run,
		BookedAt:    env.Now(),
		Corrections: correctionsRequest,
	})
	require.NoError(t, err)
	require.Len(t, corrections, 1)
	require.True(t, corrections[0].Amount.Equal(alpacadecimal.NewFromInt(-20)))

	// The correction only moves 20 ACME back into FBO out of the custom-currency
	// accrued bucket; it never creates or touches a fiat receivable/accrued/payment route.
	require.True(t, env.sumBalance(t, env.FBOSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, costBasis)).Equal(alpacadecimal.NewFromInt(20)))
	require.True(t, env.sumBalance(t, env.AccruedSubAccountForCurrency(t, customCurrencyIdentity, &settlementCurrency, &costBasis, lo.ToPtr(testChargeTaxCodeID))).Equal(alpacadecimal.NewFromInt(40)))
	require.True(t, env.sumBalance(t, env.receivableSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.invoiceAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
}

// TestCreateInitialLineages_CustomCurrency proves that credit realization
// lineage tracking - the advance/backfill/earnings-recognized provenance used
// by both credit_only and credit_then_invoice credit allocation - is currency
// agnostic and works for a custom currency the same way it does for fiat, and
// that two managed custom currencies sharing a display code (e.g. after one
// is soft-deleted and the code reused) never contaminate each other's
// lineage: loading and backfilling are scoped by managed currency ID, not
// code alone.
func TestCreateInitialLineages_CustomCurrency(t *testing.T) {
	env := newUsageBasedHandlerTestEnv(t)
	firstCurrency := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	secondCurrency := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	require.Equal(t, firstCurrency.GetCode(), secondCurrency.GetCode())
	require.NotEqual(t, firstCurrency.ID, secondCurrency.ID)

	firstChargeID := ulid.Make().String()
	secondChargeID := ulid.Make().String()
	env.ensureCharge(t, firstChargeID)
	env.ensureCharge(t, secondChargeID)

	firstRealizationID := env.createAdvanceLineage(t, firstChargeID, firstCurrency, alpacadecimal.NewFromInt(30))
	secondRealizationID := env.createAdvanceLineage(t, secondChargeID, secondCurrency, alpacadecimal.NewFromInt(50))

	firstLineages, err := env.lineage.LoadLineagesByCustomer(t.Context(), lineage.LoadLineagesByCustomerInput{
		Namespace:  env.Namespace,
		CustomerID: env.CustomerID.ID,
		Currency:   firstCurrency.Reference(),
	})
	require.NoError(t, err)
	require.Len(t, firstLineages, 1)
	require.True(t, firstLineages[0].Currency.Equal(firstCurrency.Reference()))
	require.Equal(t, firstChargeID, firstLineages[0].ChargeID)

	secondLineages, err := env.lineage.LoadLineagesByCustomer(t.Context(), lineage.LoadLineagesByCustomerInput{
		Namespace:  env.Namespace,
		CustomerID: env.CustomerID.ID,
		Currency:   secondCurrency.Reference(),
	})
	require.NoError(t, err)
	require.Len(t, secondLineages, 1)
	require.True(t, secondLineages[0].Currency.Equal(secondCurrency.Reference()))
	require.Equal(t, secondChargeID, secondLineages[0].ChargeID)

	// Backfilling the first managed currency's uncovered advance must not touch
	// the second managed currency's lineage, even though both share "ACME".
	err = env.lineage.BackfillAdvanceLineageSegments(t.Context(), lineage.BackfillAdvanceLineageSegmentsInput{
		Namespace:                 env.Namespace,
		CustomerID:                env.CustomerID.ID,
		Currency:                  firstCurrency,
		Amount:                    alpacadecimal.NewFromInt(30),
		BackingTransactionGroupID: ulid.Make().String(),
	})
	require.NoError(t, err)

	firstSegments, err := env.lineage.LoadActiveSegmentsByRealizationID(t.Context(), env.Namespace, []string{firstRealizationID})
	require.NoError(t, err)
	require.Len(t, firstSegments[firstRealizationID], 1)
	require.Equal(t, creditrealization.LineageSegmentStateAdvanceBackfilled, firstSegments[firstRealizationID][0].State)

	secondSegments, err := env.lineage.LoadActiveSegmentsByRealizationID(t.Context(), env.Namespace, []string{secondRealizationID})
	require.NoError(t, err)
	require.Len(t, secondSegments[secondRealizationID], 1)
	require.Equal(t, creditrealization.LineageSegmentStateAdvanceUncovered, secondSegments[secondRealizationID][0].State,
		"backfilling the first managed currency must not touch the second managed currency's lineage")
}

func (e *usageBasedHandlerTestEnv) createAdvanceLineage(t *testing.T, chargeID string, currency currencies.Currency, amount alpacadecimal.Decimal) string {
	t.Helper()

	now := time.Now().UTC()
	realizationID := ulid.Make().String()
	realizations := creditrealization.Realizations{
		{
			NamespacedModel: models.NamespacedModel{Namespace: e.Namespace},
			ManagedModel:    models.ManagedModel{CreatedAt: now, UpdatedAt: now},
			CreateInput: creditrealization.CreateInput{
				ID:                realizationID,
				Annotations:       creditrealization.LineageAnnotations(creditrealization.LineageOriginKindAdvance),
				ServicePeriod:     timeutil.ClosedPeriod{From: now.Add(-time.Hour), To: now},
				LedgerTransaction: ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()},
				Amount:            amount,
				Type:              creditrealization.TypeAllocation,
			},
		},
	}

	err := e.lineage.CreateInitialLineages(t.Context(), lineage.CreateInitialLineagesInput{
		Namespace:    e.Namespace,
		ChargeID:     chargeID,
		CustomerID:   e.CustomerID.ID,
		Currency:     currency,
		Realizations: realizations,
	})
	require.NoError(t, err)

	return realizationID
}

func (e *usageBasedHandlerTestEnv) newCustomCurrencyCreditsOnlyCharge(t *testing.T, customCurrencyValue currencies.Currency) chargeusagebased.Charge {
	t.Helper()

	now := time.Now().UTC()
	featureID := "feature-api-requests"
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-time.Hour),
		To:   now,
	}

	return chargeusagebased.Charge{
		ChargeBase: chargeusagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: e.Namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				ID: "usage-based-charge-cc",
			},
			Intent: chargeusagebased.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.SystemManagedLine,
					CustomerID: e.CustomerID.ID,
					Currency:   customCurrencyValue,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: testChargeTaxCodeID,
					},
				},
				IntentMutableFields: chargeusagebased.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:          "Usage based (custom currency)",
						ServicePeriod: servicePeriod,
						BillingPeriod: servicePeriod,
					},
					InvoiceAt: now,
					Price:     *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				},
				FeatureKey:     "api_requests",
				SettlementMode: productcatalog.CreditOnlySettlementMode,
			}.AsOverridableIntent(),
			Status: chargeusagebased.StatusActiveRealizationProcessing,
			State: chargeusagebased.State{
				FeatureID:    featureID,
				RatingEngine: chargeusagebased.RatingEngineDelta,
			},
		},
	}
}

// newCustomCurrencyCreditThenInvoiceCharge builds a credit_then_invoice charge
// with a manually resolved and persisted cost basis, the shape a real charge
// has once ResolveDynamicCostBasis has run and snapshotted a rate.
func (e *usageBasedHandlerTestEnv) newCustomCurrencyCreditThenInvoiceCharge(t *testing.T, customCurrencyValue currencies.Currency, costBasis alpacadecimal.Decimal) chargeusagebased.Charge {
	t.Helper()

	usd, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	now := time.Now().UTC()
	featureID := "feature-api-requests"
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-time.Hour),
		To:   now,
	}
	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: usd,
		Rate:         costBasis,
	})

	return chargeusagebased.Charge{
		ChargeBase: chargeusagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: e.Namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				ID: "usage-based-charge-cc-cti",
			},
			Intent: chargeusagebased.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.SystemManagedLine,
					CustomerID: e.CustomerID.ID,
					Currency:   customCurrencyValue,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: testChargeTaxCodeID,
					},
				},
				IntentMutableFields: chargeusagebased.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:          "Usage based (custom currency, credit then invoice)",
						ServicePeriod: servicePeriod,
						BillingPeriod: servicePeriod,
					},
					InvoiceAt: now,
					Price:     *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				},
				FeatureKey:     "api_requests",
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				CostBasis:      &costBasisIntent,
			}.AsOverridableIntent(),
			Status: chargeusagebased.StatusActiveRealizationProcessing,
			State: chargeusagebased.State{
				FeatureID:    featureID,
				RatingEngine: chargeusagebased.RatingEngineDelta,
				CostBasisID:  lo.ToPtr("cost-basis-1"),
				ResolvedCostBasis: &costbasis.State{
					CostBasis:  costBasis,
					ResolvedAt: now,
				},
			},
		},
	}
}

func (e *usageBasedHandlerTestEnv) newCustomOverageRun(overageTotals totals.Totals) chargeusagebased.RealizationRun {
	run := e.newRun()
	run.Totals = overageTotals
	return run
}

// fundCustomFBO books a known-cost-basis, fiat-sourced custom currency FBO
// balance the way a credit purchase issues one: crediting FBO and debiting an
// open receivable in the same posting. The receivable side (and its later
// fiat conversion/settlement) is irrelevant to credit_only collection, which
// only reads FBO/accrued balances, so it is left open here.
func (e *usageBasedHandlerTestEnv) fundCustomFBO(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference, costBasisCurrency *currencyx.Code, costBasis alpacadecimal.Decimal, amount alpacadecimal.Decimal) {
	t.Helper()

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService: e.Deps.ResolversService,
			AccountCatalog: e.Deps.AccountService,
			BalanceQuerier: e.Deps.HistoricalLedger,
		},
		transactions.ResolutionScope{
			CustomerID: e.CustomerID,
			Namespace:  e.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:                e.Now(),
			Amount:            amount,
			Currency:          customCurrency,
			CostBasisCurrency: costBasisCurrency,
			CostBasis:         &costBasis,
		},
	)
	require.NoError(t, err)

	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}

func (e *usageBasedHandlerTestEnv) customUnknownAccruedSubAccountForUsageBased(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference) ledger.SubAccount {
	t.Helper()

	return e.AccruedSubAccountForCurrency(t, customCurrency, nil, nil, lo.ToPtr(testChargeTaxCodeID))
}

func (e *usageBasedHandlerTestEnv) customUnknownFboSubAccountForUsageBased(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       customCurrency,
		CreditPriority: ledger.DefaultCustomerFBOPriority,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *usageBasedHandlerTestEnv) customReceivableSubAccountForUsageBasedFeature(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference, featureKey string) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       customCurrency,
		CostBasis:                      nil,
		Features:                       []string{featureKey},
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	require.NoError(t, err)

	return subAccount
}
