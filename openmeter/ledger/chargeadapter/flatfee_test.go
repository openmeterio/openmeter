package chargeadapter_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargeflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	ledgertransactiongroupdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransactiongroup"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	ledgercollector "github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestOnFlatFeeAssignedToInvoice(t *testing.T) {
	t.Run("invoice only is a no-op", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		priorityOne := env.fundPriority(t, 1, 100)

		realizations, err := env.handler.OnAssignedToInvoice(t.Context(), env.newAssignmentInputWithMode(alpacadecimal.NewFromInt(60), productcatalog.InvoiceOnlySettlementMode))
		require.NoError(t, err)
		require.Nil(t, realizations)

		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(100)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
	})

	t.Run("credit_then_invoice single bucket full coverage lands in accrued", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		priorityOne := env.fundPriority(t, 1, 100)
		input := env.newAssignmentInput(alpacadecimal.NewFromInt(60))

		realizations, err := env.handler.OnAssignedToInvoice(t.Context(), input)
		require.NoError(t, err)
		require.Len(t, realizations, 1)
		require.True(t, realizations[0].Amount.Equal(alpacadecimal.NewFromInt(60)))
		require.NotEmpty(t, realizations[0].LedgerTransaction.TransactionGroupID)
		require.Equal(
			t,
			ledger.ChargeAnnotations(models.NamespacedID{Namespace: env.Namespace, ID: input.Charge.ID}),
			env.transactionGroupAnnotations(t, realizations[0].LedgerTransaction.TransactionGroupID),
		)

		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(40)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(60)))
		require.True(t, env.sumBalance(t, env.creditEarningsSubAccount(t)).Equal(alpacadecimal.NewFromInt(0)))
	})

	t.Run("credit_then_invoice multiple buckets honor ascending priority", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		priorityTwo := env.fundPriority(t, 2, 50)
		priorityOne := env.fundPriority(t, 1, 30)

		realizations, err := env.handler.OnAssignedToInvoice(t.Context(), env.newAssignmentInput(alpacadecimal.NewFromInt(60)))
		require.NoError(t, err)
		require.Len(t, realizations, 2)
		require.True(t, realizations[0].Amount.Equal(alpacadecimal.NewFromInt(30)))
		require.True(t, realizations[1].Amount.Equal(alpacadecimal.NewFromInt(30)))
		require.NotEmpty(t, realizations[0].LedgerTransaction.TransactionGroupID)
		require.Equal(t, realizations[0].LedgerTransaction.TransactionGroupID, realizations[1].LedgerTransaction.TransactionGroupID)

		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(0)))
		require.True(t, env.sumBalance(t, priorityTwo).Equal(alpacadecimal.NewFromInt(20)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(60)))
	})

	t.Run("credit_then_invoice insufficient balance returns partial realization", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		priorityOne := env.fundPriority(t, 1, 10)
		priorityTwo := env.fundPriority(t, 2, 5)

		realizations, err := env.handler.OnAssignedToInvoice(t.Context(), env.newAssignmentInput(alpacadecimal.NewFromInt(30)))
		require.NoError(t, err)
		require.Len(t, realizations, 2)
		require.True(t, realizations[0].Amount.Equal(alpacadecimal.NewFromInt(10)))
		require.True(t, realizations[1].Amount.Equal(alpacadecimal.NewFromInt(5)))
		require.NotEmpty(t, realizations[0].LedgerTransaction.TransactionGroupID)
		require.Equal(t, realizations[0].LedgerTransaction.TransactionGroupID, realizations[1].LedgerTransaction.TransactionGroupID)

		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(0)))
		require.True(t, env.sumBalance(t, priorityTwo).Equal(alpacadecimal.NewFromInt(0)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(15)))
	})

	t.Run("credit_then_invoice zero amount returns no realization", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		priorityOne := env.fundPriority(t, 1, 100)

		realizations, err := env.handler.OnAssignedToInvoice(t.Context(), env.newAssignmentInput(alpacadecimal.NewFromInt(0)))
		require.NoError(t, err)
		require.Nil(t, realizations)
		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(100)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(0)))
	})

	t.Run("credit_only returns an error from invoice assignment handler", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		realizations, err := env.handler.OnAssignedToInvoice(t.Context(), env.newAssignmentInputWithMode(alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode))
		require.Error(t, err)
		require.Nil(t, realizations)
	})
}

func TestOnFlatFeeCreditsOnlyUsageAccrued(t *testing.T) {
	t.Run("credit_only advances uncovered amount", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		realizations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeflatfee.OnCreditsOnlyUsageAccruedInput{
			Charge:           env.newCreditsOnlyCharge(alpacadecimal.NewFromInt(30)),
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.NoError(t, err)
		require.Len(t, realizations, 1)
		require.True(t, realizations[0].Amount.Equal(alpacadecimal.NewFromInt(30)))

		require.True(t, env.sumBalance(t, env.unknownReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-30)))
		require.True(t, env.sumBalance(t, env.unknownFboSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.unknownAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(30)))
	})

	t.Run("credit_only zero amount returns no realization", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		realizations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeflatfee.OnCreditsOnlyUsageAccruedInput{
			Charge:           env.newCreditsOnlyCharge(alpacadecimal.NewFromInt(30)),
			AmountToAllocate: alpacadecimal.Zero,
		})
		require.NoError(t, err)
		require.Nil(t, realizations)
	})

	t.Run("credit_then_invoice returns an error from credits-only accrual handler", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		realizations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeflatfee.OnCreditsOnlyUsageAccruedInput{
			Charge:           env.newBaseCharge(timeutil.ClosedPeriod{From: env.Now().Add(-time.Hour), To: env.Now()}, alpacadecimal.NewFromInt(30)),
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.Error(t, err)
		require.Nil(t, realizations)
	})
}

func TestOnFlatFeeCreditsOnlyUsageAccruedCorrection(t *testing.T) {
	t.Run("credit_only reverses advance-backed accrual", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		charge := env.newCreditsOnlyCharge(alpacadecimal.NewFromInt(30))
		allocations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeflatfee.OnCreditsOnlyUsageAccruedInput{
			Charge:           charge,
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.NoError(t, err)
		require.Len(t, allocations, 1)

		chargeWithRealizations := env.newChargeWithCreditRealizationsAndAccruedUsage(allocations, alpacadecimal.Zero)
		chargeWithRealizations.Intent.SettlementMode = productcatalog.CreditOnlySettlementMode

		currencyCalculator, err := chargeWithRealizations.Intent.Currency.Calculator()
		require.NoError(t, err)

		correctionsRequest, err := chargeWithRealizations.Realizations.CreditRealizations.CreateCorrectionRequest(alpacadecimal.NewFromInt(-30), currencyCalculator)
		require.NoError(t, err)

		corrections, err := env.handler.OnCreditsOnlyUsageAccruedCorrection(t.Context(), chargeflatfee.CreditsOnlyUsageAccruedCorrectionInput{
			Charge:      chargeWithRealizations,
			AllocateAt:  env.Now(),
			Corrections: correctionsRequest,
		})
		require.NoError(t, err)
		require.Len(t, corrections, 1)
		require.True(t, corrections[0].Amount.Equal(alpacadecimal.NewFromInt(-30)))

		require.True(t, env.sumBalance(t, env.unknownReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.unknownFboSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.unknownAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
	})

	t.Run("credit_only reverses partial funded accrual in reverse priority order", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		priorityTwo := env.fundPriority(t, 2, 20)
		priorityOne := env.fundPriority(t, 1, 30)

		charge := env.newCreditsOnlyCharge(alpacadecimal.NewFromInt(50))
		allocations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeflatfee.OnCreditsOnlyUsageAccruedInput{
			Charge:           charge,
			AmountToAllocate: alpacadecimal.NewFromInt(50),
		})
		require.NoError(t, err)
		require.Len(t, allocations, 2)

		chargeWithRealizations := env.newChargeWithCreditRealizationsAndAccruedUsage(allocations, alpacadecimal.Zero)
		chargeWithRealizations.Intent.SettlementMode = productcatalog.CreditOnlySettlementMode

		currencyCalculator, err := chargeWithRealizations.Intent.Currency.Calculator()
		require.NoError(t, err)

		correctionsRequest, err := chargeWithRealizations.Realizations.CreditRealizations.CreateCorrectionRequest(alpacadecimal.NewFromInt(-35), currencyCalculator)
		require.NoError(t, err)

		corrections, err := env.handler.OnCreditsOnlyUsageAccruedCorrection(t.Context(), chargeflatfee.CreditsOnlyUsageAccruedCorrectionInput{
			Charge:      chargeWithRealizations,
			AllocateAt:  env.Now(),
			Corrections: correctionsRequest,
		})
		require.NoError(t, err)
		require.Len(t, corrections, 2)

		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(15)))
		require.True(t, env.sumBalance(t, priorityTwo).Equal(alpacadecimal.NewFromInt(20)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(15)))
	})

	t.Run("credit_only mixed funded and advance correction only unwinds the advance companion once", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		priorityOne := env.fundPriority(t, 1, 20)

		charge := env.newCreditsOnlyCharge(alpacadecimal.NewFromInt(30))
		allocations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeflatfee.OnCreditsOnlyUsageAccruedInput{
			Charge:           charge,
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.NoError(t, err)
		require.Len(t, allocations, 2)

		chargeWithRealizations := env.newChargeWithCreditRealizationsAndAccruedUsage(allocations, alpacadecimal.Zero)
		chargeWithRealizations.Intent.SettlementMode = productcatalog.CreditOnlySettlementMode

		currencyCalculator, err := chargeWithRealizations.Intent.Currency.Calculator()
		require.NoError(t, err)

		correctionsRequest, err := chargeWithRealizations.Realizations.CreditRealizations.CreateCorrectionRequest(alpacadecimal.NewFromInt(-30), currencyCalculator)
		require.NoError(t, err)

		corrections, err := env.handler.OnCreditsOnlyUsageAccruedCorrection(t.Context(), chargeflatfee.CreditsOnlyUsageAccruedCorrectionInput{
			Charge:      chargeWithRealizations,
			AllocateAt:  env.Now(),
			Corrections: correctionsRequest,
		})
		require.NoError(t, err)
		require.Len(t, corrections, 2)

		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(20)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.unknownAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.unknownReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.unknownFboSubAccount(t)).Equal(alpacadecimal.Zero))
	})

	t.Run("credit_then_invoice reverses credit-backed accrual", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)
		priorityOne := env.fundPriority(t, 1, 30)

		allocations, err := env.handler.OnAssignedToInvoice(t.Context(), env.newAssignmentInput(alpacadecimal.NewFromInt(30)))
		require.NoError(t, err)
		require.Len(t, allocations, 1)

		chargeWithRealizations := env.newChargeWithCreditRealizationsAndAccruedUsage(allocations, alpacadecimal.Zero)

		currencyCalculator, err := chargeWithRealizations.Intent.Currency.Calculator()
		require.NoError(t, err)

		correctionsRequest, err := chargeWithRealizations.Realizations.CreditRealizations.CreateCorrectionRequest(alpacadecimal.NewFromInt(-30), currencyCalculator)
		require.NoError(t, err)

		corrections, err := env.handler.OnCreditsOnlyUsageAccruedCorrection(t.Context(), chargeflatfee.CreditsOnlyUsageAccruedCorrectionInput{
			Charge:      chargeWithRealizations,
			AllocateAt:  env.Now(),
			Corrections: correctionsRequest,
		})
		require.NoError(t, err)
		require.Len(t, corrections, 1)
		require.True(t, corrections[0].Amount.Equal(alpacadecimal.NewFromInt(-30)))

		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(30)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
	})

	t.Run("credit_then_invoice reverses recognized earnings in the same correction", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)
		priorityOne := env.fundPriority(t, 1, 30)

		allocations, err := env.handler.OnAssignedToInvoice(t.Context(), env.newAssignmentInput(alpacadecimal.NewFromInt(30)))
		require.NoError(t, err)
		require.Len(t, allocations, 1)

		chargeWithRealizations := env.newChargeWithCreditRealizationsAndAccruedUsage(allocations, alpacadecimal.Zero)
		env.createInitialLineages(t, chargeWithRealizations.ID, chargeWithRealizations.Realizations.CreditRealizations)
		recognitionGroupID := env.recognizeCreditAccrued(t, alpacadecimal.NewFromInt(30))

		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.creditEarningsSubAccount(t)).Equal(alpacadecimal.NewFromInt(30)))

		currencyCalculator, err := chargeWithRealizations.Intent.Currency.Calculator()
		require.NoError(t, err)

		correctionsRequest, err := chargeWithRealizations.Realizations.CreditRealizations.CreateCorrectionRequest(alpacadecimal.NewFromInt(-30), currencyCalculator)
		require.NoError(t, err)

		corrections, err := env.handler.OnCreditsOnlyUsageAccruedCorrection(t.Context(), chargeflatfee.CreditsOnlyUsageAccruedCorrectionInput{
			Charge:                       chargeWithRealizations,
			AllocateAt:                   env.Now(),
			Corrections:                  correctionsRequest,
			LineageSegmentsByRealization: env.assertRecognizedSegments(t, chargeWithRealizations.Realizations.CreditRealizations, recognitionGroupID),
		})
		require.NoError(t, err)
		require.Len(t, corrections, 1)
		require.True(t, corrections[0].Amount.Equal(alpacadecimal.NewFromInt(-30)))

		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(30)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.creditEarningsSubAccount(t)).Equal(alpacadecimal.Zero))
	})

	t.Run("credit_only reverses recognized earnings in the same correction", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)
		priorityOne := env.fundPriority(t, 1, 30)

		charge := env.newCreditsOnlyCharge(alpacadecimal.NewFromInt(30))
		allocations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeflatfee.OnCreditsOnlyUsageAccruedInput{
			Charge:           charge,
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.NoError(t, err)
		require.Len(t, allocations, 1)

		chargeWithRealizations := env.newChargeWithCreditRealizationsAndAccruedUsage(allocations, alpacadecimal.Zero)
		chargeWithRealizations.Intent.SettlementMode = productcatalog.CreditOnlySettlementMode
		env.createInitialLineages(t, chargeWithRealizations.ID, chargeWithRealizations.Realizations.CreditRealizations)
		recognitionGroupID := env.recognizeCreditAccrued(t, alpacadecimal.NewFromInt(30))

		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.creditEarningsSubAccount(t)).Equal(alpacadecimal.NewFromInt(30)))

		currencyCalculator, err := chargeWithRealizations.Intent.Currency.Calculator()
		require.NoError(t, err)

		correctionsRequest, err := chargeWithRealizations.Realizations.CreditRealizations.CreateCorrectionRequest(alpacadecimal.NewFromInt(-30), currencyCalculator)
		require.NoError(t, err)

		corrections, err := env.handler.OnCreditsOnlyUsageAccruedCorrection(t.Context(), chargeflatfee.CreditsOnlyUsageAccruedCorrectionInput{
			Charge:                       chargeWithRealizations,
			AllocateAt:                   env.Now(),
			Corrections:                  correctionsRequest,
			LineageSegmentsByRealization: env.assertRecognizedSegments(t, chargeWithRealizations.Realizations.CreditRealizations, recognitionGroupID),
		})
		require.NoError(t, err)
		require.Len(t, corrections, 1)
		require.True(t, corrections[0].Amount.Equal(alpacadecimal.NewFromInt(-30)))

		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(30)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.creditEarningsSubAccount(t)).Equal(alpacadecimal.Zero))
	})
}

func TestOnFlatFeeStandardInvoiceUsageAccrued(t *testing.T) {
	t.Run("credit_then_invoice books receivable-backed usage into accrued", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		total := alpacadecimal.NewFromInt(50)
		ref, err := env.handler.OnInvoiceUsageAccrued(t.Context(), env.newAccrualInput(total))
		require.NoError(t, err)
		require.NotEmpty(t, ref.TransactionGroupID)

		require.True(t, env.sumBalance(t, env.receivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-50)))
		require.True(t, env.sumBalance(t, env.invoiceAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(50)))
	})

	t.Run("credit_then_invoice zero total returns empty reference", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		ref, err := env.handler.OnInvoiceUsageAccrued(t.Context(), env.newAccrualInput(alpacadecimal.NewFromInt(0)))
		require.NoError(t, err)
		require.Empty(t, ref.TransactionGroupID)
	})

	t.Run("credit_only returns an error from invoice accrual handler", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		input := env.newAccrualInput(alpacadecimal.NewFromInt(10))
		input.Charge.Intent.SettlementMode = productcatalog.CreditOnlySettlementMode

		ref, err := env.handler.OnInvoiceUsageAccrued(t.Context(), input)
		require.Error(t, err)
		require.Empty(t, ref.TransactionGroupID)
	})
}

func TestOnFlatFeePaymentAuthorized(t *testing.T) {
	t.Run("credit_then_invoice stages open receivable as authorized", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		// First accrue usage: receivable → accrued
		total := alpacadecimal.NewFromInt(75)
		_, err := env.handler.OnInvoiceUsageAccrued(t.Context(), env.newAccrualInput(total))
		require.NoError(t, err)

		charge := env.newChargeWithAccruedUsage(total)
		ref, err := env.handler.OnPaymentAuthorized(t.Context(), charge)
		require.NoError(t, err)
		require.NotEmpty(t, ref.TransactionGroupID)

		// Authorization only moves the receivable between status buckets.
		require.True(t, env.sumBalance(t, env.receivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-75)))
		require.True(t, env.sumBalance(t, env.washSubAccount(t)).Equal(alpacadecimal.Zero))
		// No revenue recognition happens here anymore.
		require.True(t, env.sumBalance(t, env.invoiceAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(75)))
		require.True(t, env.sumBalance(t, env.invoiceEarningsSubAccount(t)).Equal(alpacadecimal.Zero))
	})

	t.Run("credit_then_invoice mixed FBO and receivable only authorizes receivable", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		// Fund FBO with 40
		env.fundPriority(t, 1, 40)

		// FBO → accrued for 40
		realizations, err := env.handler.OnAssignedToInvoice(t.Context(), env.newAssignmentInput(alpacadecimal.NewFromInt(60)))
		require.NoError(t, err)
		require.Len(t, realizations, 1)
		require.True(t, realizations[0].Amount.Equal(alpacadecimal.NewFromInt(40)))

		// Remaining 20: receivable → accrued
		remaining := alpacadecimal.NewFromInt(20)
		_, err = env.handler.OnInvoiceUsageAccrued(t.Context(), env.newAccrualInput(remaining))
		require.NoError(t, err)

		// Accrued is now split by cost basis: 40 credit-backed (0) + 20 invoice-backed (1).
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(40)))
		require.True(t, env.sumBalance(t, env.invoiceAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(20)))

		// Authorization with both credit realizations and accrued usage
		charge := env.newChargeWithCreditRealizationsAndAccruedUsage(realizations, remaining)
		ref, err := env.handler.OnPaymentAuthorized(t.Context(), charge)
		require.NoError(t, err)
		require.NotEmpty(t, ref.TransactionGroupID)

		// Cash movement stays deferred until settlement.
		require.True(t, env.sumBalance(t, env.receivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-20)))
		// Existing accrued balances stay untouched until a later recognition flow.
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(40)))
		require.True(t, env.sumBalance(t, env.invoiceAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(20)))
		require.True(t, env.sumBalance(t, env.creditEarningsSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.invoiceEarningsSubAccount(t)).Equal(alpacadecimal.Zero))
	})

	t.Run("credit_then_invoice does not touch accrued during authorization", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		_, err := env.handler.OnInvoiceUsageAccrued(t.Context(), env.newAccrualInput(alpacadecimal.NewFromInt(30)))
		require.NoError(t, err)

		charge := env.newChargeWithAccruedUsage(alpacadecimal.NewFromInt(75))
		ref, err := env.handler.OnPaymentAuthorized(t.Context(), charge)
		require.NoError(t, err)
		require.NotEmpty(t, ref.TransactionGroupID)

		require.True(t, env.sumBalance(t, env.receivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(45)))
		require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-75)))
		require.True(t, env.sumBalance(t, env.washSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.invoiceAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(30)))
		require.True(t, env.sumBalance(t, env.invoiceEarningsSubAccount(t)).Equal(alpacadecimal.Zero))
	})

	t.Run("credit_only authorization is a no-op without receivable funding", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		priorityOne := env.fundPriority(t, 1, 40)

		realizations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeflatfee.OnCreditsOnlyUsageAccruedInput{
			Charge:           env.newCreditsOnlyCharge(alpacadecimal.NewFromInt(30)),
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.NoError(t, err)
		require.Len(t, realizations, 1)

		charge := env.newChargeWithCreditRealizationsAndAccruedUsage(realizations, alpacadecimal.Zero)
		charge.Intent.SettlementMode = productcatalog.CreditOnlySettlementMode

		ref, err := env.handler.OnPaymentAuthorized(t.Context(), charge)
		require.NoError(t, err)
		require.Empty(t, ref.TransactionGroupID)

		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(30)))
		require.True(t, env.sumBalance(t, env.creditEarningsSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(10)))
		require.True(t, env.sumBalance(t, env.receivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.invoiceEarningsSubAccount(t)).Equal(alpacadecimal.Zero))
	})
}

func TestOnFlatFeePaymentSettled(t *testing.T) {
	t.Run("credit_then_invoice settles authorized receivable from wash", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		total := alpacadecimal.NewFromInt(40)
		_, err := env.handler.OnInvoiceUsageAccrued(t.Context(), env.newAccrualInput(total))
		require.NoError(t, err)

		charge := env.newChargeWithAccruedUsage(total)
		_, err = env.handler.OnPaymentAuthorized(t.Context(), charge)
		require.NoError(t, err)

		ref, err := env.handler.OnPaymentSettled(t.Context(), charge)
		require.NoError(t, err)
		require.NotEmpty(t, ref.TransactionGroupID)

		require.True(t, env.sumBalance(t, env.receivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.washSubAccount(t)).Equal(alpacadecimal.NewFromInt(-40)))
	})

	t.Run("credit_then_invoice no receivable portion is a no-op", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		ref, err := env.handler.OnPaymentSettled(t.Context(), env.newChargeWithCreditRealizationsAndAccruedUsage(nil, alpacadecimal.Zero))
		require.NoError(t, err)
		require.Empty(t, ref.TransactionGroupID)
	})

	t.Run("credit_only payment settled is a no-op after authorization", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		env.fundPriority(t, 1, 30)

		realizations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeflatfee.OnCreditsOnlyUsageAccruedInput{
			Charge:           env.newCreditsOnlyCharge(alpacadecimal.NewFromInt(30)),
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.NoError(t, err)

		charge := env.newChargeWithCreditRealizationsAndAccruedUsage(realizations, alpacadecimal.Zero)
		charge.Intent.SettlementMode = productcatalog.CreditOnlySettlementMode

		_, err = env.handler.OnPaymentAuthorized(t.Context(), charge)
		require.NoError(t, err)

		ref, err := env.handler.OnPaymentSettled(t.Context(), charge)
		require.NoError(t, err)
		require.Empty(t, ref.TransactionGroupID)
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(30)))
		require.True(t, env.sumBalance(t, env.creditEarningsSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.receivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
	})
}

func TestOnFlatFeePaymentUncollectible(t *testing.T) {
	t.Run("returns descriptive error", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		charge := env.newChargeWithAccruedUsage(alpacadecimal.NewFromInt(30))
		_, err := env.handler.OnPaymentUncollectible(t.Context(), charge)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not yet implemented")
	})
}

type flatFeeHandlerTestEnv struct {
	*ledgertestutils.IntegrationEnv
	handler    chargeflatfee.Handler
	lineage    lineage.Service
	recognizer recognizer.Service
}

func newFlatFeeHandlerTestEnv(t *testing.T) *flatFeeHandlerTestEnv {
	base := ledgertestutils.NewIntegrationEnv(t, "chargeadapter-flatfee")
	deps := transactions.ResolverDependencies{
		AccountService:    base.Deps.ResolversService,
		SubAccountService: base.Deps.AccountService,
	}
	collectorService := ledgercollector.NewService(ledgercollector.Config{
		Ledger:       base.Deps.HistoricalLedger,
		Dependencies: deps,
	})
	lineageAdapter, err := lineageadapter.New(lineageadapter.Config{
		Client: base.DB,
	})
	require.NoError(t, err)

	lineageService, err := lineageservice.New(lineageservice.Config{
		Adapter: lineageAdapter,
	})
	require.NoError(t, err)

	recognizerService, err := recognizer.NewService(recognizer.Config{
		Ledger:             base.Deps.HistoricalLedger,
		Dependencies:       deps,
		Lineage:            lineageService,
		TransactionManager: enttx.NewCreator(base.DB),
	})
	require.NoError(t, err)

	return &flatFeeHandlerTestEnv{
		IntegrationEnv: base,
		handler: chargeadapter.NewFlatFeeHandler(
			base.Deps.HistoricalLedger,
			deps,
			collectorService,
		),
		lineage:    lineageService,
		recognizer: recognizerService,
	}
}

func (e *flatFeeHandlerTestEnv) newAssignmentInput(amount alpacadecimal.Decimal) chargeflatfee.OnAssignedToInvoiceInput {
	return e.newAssignmentInputWithMode(amount, productcatalog.CreditThenInvoiceSettlementMode)
}

func (e *flatFeeHandlerTestEnv) newAssignmentInputWithMode(amount alpacadecimal.Decimal, mode productcatalog.SettlementMode) chargeflatfee.OnAssignedToInvoiceInput {
	now := time.Now().UTC()
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-time.Hour),
		To:   now,
	}

	return chargeflatfee.OnAssignedToInvoiceInput{
		Charge: chargeflatfee.Charge{
			ChargeBase: chargeflatfee.ChargeBase{
				ManagedResource: meta.ManagedResource{
					NamespacedModel: models.NamespacedModel{
						Namespace: e.Namespace,
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: now,
						UpdatedAt: now,
					},
					ID: "flat-fee-charge",
				},
				Intent: chargeflatfee.Intent{
					Intent: meta.Intent{
						Name:              "Flat fee",
						ManagedBy:         billing.SystemManagedLine,
						CustomerID:        e.CustomerID.ID,
						Currency:          currencyx.Code("USD"),
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:             now,
					SettlementMode:        mode,
					PaymentTerm:           productcatalog.InAdvancePaymentTerm,
					ProRating:             productcatalog.ProRatingConfig{},
					AmountBeforeProration: amount,
				},
				State: chargeflatfee.State{
					AmountAfterProration: amount,
				},
				Status: chargeflatfee.StatusActive,
			},
		},
		ServicePeriod:     servicePeriod,
		PreTaxTotalAmount: amount,
	}
}

func (e *flatFeeHandlerTestEnv) fundPriority(t *testing.T, priority int, amount int64) ledger.SubAccount {
	t.Helper()

	costBasis := alpacadecimal.Zero
	subAccount, err := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       e.Currency,
		CostBasis:      &costBasis,
		CreditPriority: priority,
	})
	require.NoError(t, err)

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService:    e.Deps.ResolversService,
			SubAccountService: e.Deps.AccountService,
		},
		transactions.ResolutionScope{
			CustomerID: e.CustomerID,
			Namespace:  e.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:             e.Now(),
			Amount:         alpacadecimal.NewFromInt(amount),
			Currency:       e.Currency,
			CostBasis:      &costBasis,
			CreditPriority: &priority,
		},
		transactions.AuthorizeCustomerReceivablePaymentTemplate{
			At:        e.Now(),
			Amount:    alpacadecimal.NewFromInt(amount),
			Currency:  e.Currency,
			CostBasis: &costBasis,
		},
		transactions.SettleCustomerReceivableFromPaymentTemplate{
			At:        e.Now(),
			Amount:    alpacadecimal.NewFromInt(amount),
			Currency:  e.Currency,
			CostBasis: &costBasis,
		},
	)
	require.NoError(t, err)

	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(
		e.Namespace,
		nil,
		inputs...,
	))
	require.NoError(t, err)

	return subAccount
}

func (e *flatFeeHandlerTestEnv) newAccrualInput(total alpacadecimal.Decimal) chargeflatfee.OnInvoiceUsageAccruedInput {
	now := time.Now().UTC()
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-time.Hour),
		To:   now,
	}

	return chargeflatfee.OnInvoiceUsageAccruedInput{
		Charge:        e.newBaseCharge(servicePeriod, total),
		ServicePeriod: servicePeriod,
		Totals: totals.Totals{
			Amount: total,
			Total:  total,
		},
	}
}

func (e *flatFeeHandlerTestEnv) newCreditsOnlyCharge(amount alpacadecimal.Decimal) chargeflatfee.Charge {
	now := time.Now().UTC()
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-time.Hour),
		To:   now,
	}

	charge := e.newBaseCharge(servicePeriod, amount)
	charge.Intent.SettlementMode = productcatalog.CreditOnlySettlementMode

	return charge
}

func (e *flatFeeHandlerTestEnv) newBaseCharge(servicePeriod timeutil.ClosedPeriod, amount alpacadecimal.Decimal) chargeflatfee.Charge {
	return chargeflatfee.Charge{
		ChargeBase: chargeflatfee.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: e.Namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: servicePeriod.To,
					UpdatedAt: servicePeriod.To,
				},
				ID: "flat-fee-charge",
			},
			Intent: chargeflatfee.Intent{
				Intent: meta.Intent{
					Name:              "Flat fee",
					ManagedBy:         billing.SystemManagedLine,
					CustomerID:        e.CustomerID.ID,
					Currency:          currencyx.Code("USD"),
					ServicePeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
					BillingPeriod:     servicePeriod,
				},
				InvoiceAt:             servicePeriod.To,
				SettlementMode:        productcatalog.InvoiceOnlySettlementMode,
				PaymentTerm:           productcatalog.InAdvancePaymentTerm,
				ProRating:             productcatalog.ProRatingConfig{},
				AmountBeforeProration: amount,
			},
			State: chargeflatfee.State{
				AmountAfterProration: amount,
			},
			Status: chargeflatfee.StatusActive,
		},
	}
}

func (e *flatFeeHandlerTestEnv) newChargeWithAccruedUsage(total alpacadecimal.Decimal) chargeflatfee.Charge {
	now := time.Now().UTC()
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-time.Hour),
		To:   now,
	}

	charge := e.newBaseCharge(servicePeriod, total)
	charge.Realizations.AccruedUsage = &invoicedusage.AccruedUsage{
		ServicePeriod: servicePeriod,
		Mutable:       true,
		Totals: totals.Totals{
			Amount: total,
			Total:  total,
		},
	}

	return charge
}

func (e *flatFeeHandlerTestEnv) newChargeWithCreditRealizationsAndAccruedUsage(realizations creditrealization.CreateAllocationInputs, accruedTotal alpacadecimal.Decimal) chargeflatfee.Charge {
	now := time.Now().UTC()
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-time.Hour),
		To:   now,
	}

	// Compute total amount (FBO + receivable portions)
	totalAmount := accruedTotal
	creditRealizations := make(creditrealization.Realizations, 0, len(realizations))
	for i, r := range realizations.AsCreateInputs() {
		totalAmount = totalAmount.Add(r.Amount)
		r.ID = ulid.Make().String()
		creditRealizations = append(creditRealizations, creditrealization.Realization{
			NamespacedModel: models.NamespacedModel{
				Namespace: e.Namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			CreateInput: r,
			SortHint:    i,
		})
	}

	charge := e.newBaseCharge(servicePeriod, totalAmount)
	charge.Realizations.CreditRealizations = creditRealizations
	charge.Realizations.AccruedUsage = &invoicedusage.AccruedUsage{
		ServicePeriod: servicePeriod,
		Mutable:       true,
		Totals: totals.Totals{
			Amount: accruedTotal,
			Total:  accruedTotal,
		},
	}

	return charge
}

func (e *flatFeeHandlerTestEnv) washSubAccount(t *testing.T) ledger.SubAccount {
	return e.WashSubAccountWithCostBasis(t, testInvoiceCostBasis())
}

func (e *flatFeeHandlerTestEnv) receivableSubAccount(t *testing.T) ledger.SubAccount {
	return e.ReceivableSubAccountWithCostBasis(t, testInvoiceCostBasis())
}

func (e *flatFeeHandlerTestEnv) authorizedReceivableSubAccount(t *testing.T) ledger.SubAccount {
	return e.ReceivableSubAccountWithCostBasisAndStatus(t, testInvoiceCostBasis(), ledger.TransactionAuthorizationStatusAuthorized)
}

func (e *flatFeeHandlerTestEnv) creditAccruedSubAccount(t *testing.T) ledger.SubAccount {
	zeroCostBasis := alpacadecimal.Zero
	return e.AccruedSubAccountWithCostBasis(t, &zeroCostBasis)
}

func (e *flatFeeHandlerTestEnv) unknownAccruedSubAccount(t *testing.T) ledger.SubAccount {
	return e.AccruedSubAccountWithCostBasis(t, nil)
}

func (e *flatFeeHandlerTestEnv) unknownReceivableSubAccount(t *testing.T) ledger.SubAccount {
	return e.ReceivableSubAccountWithCostBasis(t, nil)
}

func (e *flatFeeHandlerTestEnv) unknownFboSubAccount(t *testing.T) ledger.SubAccount {
	subAccount, err := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       e.Currency,
		CreditPriority: ledger.DefaultCustomerFBOPriority,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *flatFeeHandlerTestEnv) invoiceAccruedSubAccount(t *testing.T) ledger.SubAccount {
	return e.AccruedSubAccountWithCostBasis(t, testInvoiceCostBasis())
}

func (e *flatFeeHandlerTestEnv) creditEarningsSubAccount(t *testing.T) ledger.SubAccount {
	zeroCostBasis := alpacadecimal.Zero
	return e.EarningsSubAccountWithCostBasis(t, &zeroCostBasis)
}

func (e *flatFeeHandlerTestEnv) invoiceEarningsSubAccount(t *testing.T) ledger.SubAccount {
	return e.EarningsSubAccountWithCostBasis(t, testInvoiceCostBasis())
}

func (e *flatFeeHandlerTestEnv) sumBalance(t *testing.T, subAccount ledger.SubAccount) alpacadecimal.Decimal {
	return e.SumBalance(t, subAccount)
}

func (e *flatFeeHandlerTestEnv) recognizeCreditAccrued(t *testing.T, amount alpacadecimal.Decimal) string {
	t.Helper()

	result, err := e.recognizer.RecognizeEarnings(t.Context(), recognizer.RecognizeEarningsInput{
		CustomerID: e.CustomerID,
		At:         e.Now(),
		Currency:   e.Currency,
	})
	require.NoError(t, err)
	require.True(t, result.RecognizedAmount.Equal(amount), "recognized=%s expected=%s", result.RecognizedAmount, amount)

	return result.LedgerGroupID
}

func (e *flatFeeHandlerTestEnv) createInitialLineages(t *testing.T, chargeID string, realizations creditrealization.Realizations) {
	t.Helper()

	e.ensureCharge(t, chargeID)

	err := e.lineage.CreateInitialLineages(t.Context(), lineage.CreateInitialLineagesInput{
		Namespace:    e.Namespace,
		ChargeID:     chargeID,
		CustomerID:   e.CustomerID.ID,
		Currency:     e.Currency,
		Realizations: realizations,
	})
	require.NoError(t, err)
}

func (e *flatFeeHandlerTestEnv) activeSegmentsByRealization(t *testing.T, realizations creditrealization.Realizations) lineage.ActiveSegmentsByRealizationID {
	t.Helper()

	ids := make([]string, 0, len(realizations))
	for _, realization := range realizations {
		ids = append(ids, realization.ID)
	}

	segments, err := e.lineage.LoadActiveSegmentsByRealizationID(t.Context(), e.Namespace, ids)
	require.NoError(t, err)

	return segments
}

func (e *flatFeeHandlerTestEnv) assertRecognizedSegments(t *testing.T, realizations creditrealization.Realizations, recognitionGroupID string) lineage.ActiveSegmentsByRealizationID {
	t.Helper()

	segmentsByRealization := e.activeSegmentsByRealization(t, realizations)
	for _, realization := range realizations {
		segments := segmentsByRealization[realization.ID]
		require.Len(t, segments, 1)

		segment := segments[0]
		require.Equal(t, creditrealization.LineageSegmentStateEarningsRecognized, segment.State)
		require.True(t, segment.Amount.Equal(realization.Amount), "segment=%s expected=%s", segment.Amount, realization.Amount)
		require.NotNil(t, segment.BackingTransactionGroupID)
		require.Equal(t, recognitionGroupID, *segment.BackingTransactionGroupID)
		require.NotNil(t, segment.SourceState)
		require.Equal(t, creditrealization.LineageSegmentStateRealCredit, *segment.SourceState)
		require.Nil(t, segment.SourceBackingTransactionGroupID)
	}

	return segmentsByRealization
}

func (e *flatFeeHandlerTestEnv) ensureCharge(t *testing.T, chargeID string) {
	t.Helper()

	_, err := e.DB.Charge.Create().
		SetID(chargeID).
		SetNamespace(e.Namespace).
		SetType(meta.ChargeTypeFlatFee).
		Save(t.Context())
	require.NoError(t, err)
}

func (e *flatFeeHandlerTestEnv) transactionGroupAnnotations(t *testing.T, groupID string) models.Annotations {
	t.Helper()

	group, err := e.DB.LedgerTransactionGroup.Query().
		Where(
			ledgertransactiongroupdb.Namespace(e.Namespace),
			ledgertransactiongroupdb.ID(groupID),
		).
		Only(t.Context())
	require.NoError(t, err)

	return group.Annotations
}

func testInvoiceCostBasis() *alpacadecimal.Decimal {
	value := alpacadecimal.NewFromInt(1)
	return &value
}
