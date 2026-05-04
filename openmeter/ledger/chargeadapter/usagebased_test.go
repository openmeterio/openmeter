package chargeadapter_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	chargeusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	ledgertransactiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	ledgercollector "github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestOnUsageBasedCreditsOnlyUsageAccrued(t *testing.T) {
	t.Run("credit_only advances uncovered amount", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)

		realizations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedInput{
			Charge:           env.newCreditsOnlyCharge(),
			Run:              env.newRun(),
			AllocateAt:       env.Now(),
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.NoError(t, err)
		require.Len(t, realizations, 1)
		require.True(t, realizations[0].Amount.Equal(alpacadecimal.NewFromInt(30)))

		require.True(t, env.sumBalance(t, env.unknownReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-30)))
		require.True(t, env.sumBalance(t, env.unknownFboSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.unknownAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(30)))
	})

	t.Run("credit_only collects from funded balances first", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)

		priorityOne := env.fundPriority(t, 1, 20)

		realizations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedInput{
			Charge:           env.newCreditsOnlyCharge(),
			Run:              env.newRun(),
			AllocateAt:       env.Now(),
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.NoError(t, err)
		require.Len(t, realizations, 2)
		require.True(t, realizations[0].Amount.Equal(alpacadecimal.NewFromInt(20)))
		require.True(t, realizations[1].Amount.Equal(alpacadecimal.NewFromInt(10)))

		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.unknownReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-10)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(20)))
		require.True(t, env.sumBalance(t, env.unknownAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(10)))
	})

	t.Run("credit_then_invoice collects available credits", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)

		priorityOne := env.fundPriority(t, 1, 20)

		realizations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedInput{
			Charge:           env.newCharge(productcatalog.CreditThenInvoiceSettlementMode),
			Run:              env.newRun(),
			AllocateAt:       env.Now(),
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.NoError(t, err)
		require.Len(t, realizations, 1)
		require.True(t, realizations[0].Amount.Equal(alpacadecimal.NewFromInt(20)))

		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.unknownReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(20)))
		require.True(t, env.sumBalance(t, env.unknownAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
	})

	t.Run("tracks charge references on transactions", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)

		charge := env.newCreditsOnlyCharge()
		charge.Intent.Subscription = &meta.SubscriptionReference{
			SubscriptionID: "subscription-01JABCDEF0123456789ABCDEF",
			PhaseID:        "phase-01JABCDEF0123456789ABCDEF",
			ItemID:         "item-01JABCDEF0123456789ABCDEF",
		}

		realizations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedInput{
			Charge:           charge,
			Run:              env.newRun(),
			AllocateAt:       env.Now(),
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.NoError(t, err)
		require.Len(t, realizations, 1)

		transactionAnnotations := env.transactionAnnotations(t, realizations[0].LedgerTransaction.TransactionGroupID)
		require.NotEmpty(t, transactionAnnotations)
		for _, annotations := range transactionAnnotations {
			require.Equal(t, charge.ID, annotations[ledger.AnnotationChargeID])
			require.Equal(t, env.Namespace, annotations[ledger.AnnotationChargeNamespace])
			require.Equal(t, charge.Intent.Subscription.SubscriptionID, annotations[ledger.AnnotationSubscriptionID])
			require.Equal(t, charge.Intent.Subscription.PhaseID, annotations[ledger.AnnotationSubscriptionPhaseID])
			require.Equal(t, charge.Intent.Subscription.ItemID, annotations[ledger.AnnotationSubscriptionItemID])
			require.Equal(t, charge.State.FeatureID, annotations[ledger.AnnotationFeatureID])
		}
	})

	t.Run("zero amount is rejected by input validation", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)

		realizations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedInput{
			Charge:           env.newCreditsOnlyCharge(),
			Run:              env.newRun(),
			AllocateAt:       env.Now(),
			AmountToAllocate: alpacadecimal.Zero,
		})
		require.Error(t, err)
		require.Nil(t, realizations)
		require.Contains(t, err.Error(), "amount to allocate must be positive")
	})
}

func TestOnUsageBasedCreditsOnlyUsageAccruedCorrection(t *testing.T) {
	t.Run("credit_only reverses advance-backed accrual", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)

		run := env.newRun()
		allocations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedInput{
			Charge:           env.newCreditsOnlyCharge(),
			Run:              run,
			AllocateAt:       env.Now(),
			AmountToAllocate: alpacadecimal.NewFromInt(30),
		})
		require.NoError(t, err)
		require.Len(t, allocations, 1)

		run.CreditsAllocated = env.realizationsFromAllocations(allocations)

		currencyCalculator, err := env.Currency.Calculator()
		require.NoError(t, err)

		correctionsRequest, err := run.CreditsAllocated.CreateCorrectionRequest(alpacadecimal.NewFromInt(-30), currencyCalculator)
		require.NoError(t, err)

		corrections, err := env.handler.OnCreditsOnlyUsageAccruedCorrection(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedCorrectionInput{
			Charge:      env.newCreditsOnlyCharge(),
			Run:         run,
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

	t.Run("credit_then_invoice reverses accrual", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)
		_ = env.fundPriority(t, 1, 20)

		run := env.newRun()
		allocations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedInput{
			Charge:           env.newCharge(productcatalog.CreditThenInvoiceSettlementMode),
			Run:              run,
			AllocateAt:       env.Now(),
			AmountToAllocate: alpacadecimal.NewFromInt(20),
		})
		require.NoError(t, err)
		require.Len(t, allocations, 1)

		run.CreditsAllocated = env.realizationsFromAllocations(allocations)

		currencyCalculator, err := env.Currency.Calculator()
		require.NoError(t, err)

		correctionsRequest, err := run.CreditsAllocated.CreateCorrectionRequest(alpacadecimal.NewFromInt(-20), currencyCalculator)
		require.NoError(t, err)

		corrections, err := env.handler.OnCreditsOnlyUsageAccruedCorrection(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedCorrectionInput{
			Charge:      env.newCharge(productcatalog.CreditThenInvoiceSettlementMode),
			Run:         run,
			AllocateAt:  env.Now(),
			Corrections: correctionsRequest,
		})
		require.NoError(t, err)
		require.Len(t, corrections, 1)
		require.True(t, corrections[0].Amount.Equal(alpacadecimal.NewFromInt(-20)))

		require.True(t, env.sumBalance(t, env.unknownReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.unknownFboSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.unknownAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
	})

	t.Run("credit_then_invoice reverses recognized earnings in the same correction", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)
		priorityOne := env.fundPriority(t, 1, 20)

		charge := env.newCharge(productcatalog.CreditThenInvoiceSettlementMode)
		run := env.newRun()
		allocations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedInput{
			Charge:           charge,
			Run:              run,
			AllocateAt:       env.Now(),
			AmountToAllocate: alpacadecimal.NewFromInt(20),
		})
		require.NoError(t, err)
		require.Len(t, allocations, 1)

		run.CreditsAllocated = env.realizationsFromAllocations(allocations)

		env.createInitialLineages(t, charge.ID, run.CreditsAllocated)
		recognitionGroupID := env.recognizeCreditAccrued(t, alpacadecimal.NewFromInt(20))
		zeroCostBasis := alpacadecimal.Zero
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.EarningsSubAccountWithCostBasis(t, &zeroCostBasis)).Equal(alpacadecimal.NewFromInt(20)))

		currencyCalculator, err := env.Currency.Calculator()
		require.NoError(t, err)

		correctionsRequest, err := run.CreditsAllocated.CreateCorrectionRequest(alpacadecimal.NewFromInt(-20), currencyCalculator)
		require.NoError(t, err)

		corrections, err := env.handler.OnCreditsOnlyUsageAccruedCorrection(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedCorrectionInput{
			Charge:                       charge,
			Run:                          run,
			AllocateAt:                   env.Now(),
			Corrections:                  correctionsRequest,
			LineageSegmentsByRealization: env.assertRecognizedSegments(t, run.CreditsAllocated, recognitionGroupID),
		})
		require.NoError(t, err)
		require.Len(t, corrections, 1)
		require.True(t, corrections[0].Amount.Equal(alpacadecimal.NewFromInt(-20)))

		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(20)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.EarningsSubAccountWithCostBasis(t, &zeroCostBasis)).Equal(alpacadecimal.Zero))
	})

	t.Run("credit_only reverses recognized earnings in the same correction", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)
		priorityOne := env.fundPriority(t, 1, 20)

		charge := env.newCreditsOnlyCharge()
		run := env.newRun()
		allocations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedInput{
			Charge:           charge,
			Run:              run,
			AllocateAt:       env.Now(),
			AmountToAllocate: alpacadecimal.NewFromInt(20),
		})
		require.NoError(t, err)
		require.Len(t, allocations, 1)

		run.CreditsAllocated = env.realizationsFromAllocations(allocations)

		env.createInitialLineages(t, charge.ID, run.CreditsAllocated)
		recognitionGroupID := env.recognizeCreditAccrued(t, alpacadecimal.NewFromInt(20))
		zeroCostBasis := alpacadecimal.Zero
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.EarningsSubAccountWithCostBasis(t, &zeroCostBasis)).Equal(alpacadecimal.NewFromInt(20)))

		currencyCalculator, err := env.Currency.Calculator()
		require.NoError(t, err)

		correctionsRequest, err := run.CreditsAllocated.CreateCorrectionRequest(alpacadecimal.NewFromInt(-20), currencyCalculator)
		require.NoError(t, err)

		corrections, err := env.handler.OnCreditsOnlyUsageAccruedCorrection(t.Context(), chargeusagebased.CreditsOnlyUsageAccruedCorrectionInput{
			Charge:                       charge,
			Run:                          run,
			AllocateAt:                   env.Now(),
			Corrections:                  correctionsRequest,
			LineageSegmentsByRealization: env.assertRecognizedSegments(t, run.CreditsAllocated, recognitionGroupID),
		})
		require.NoError(t, err)
		require.Len(t, corrections, 1)
		require.True(t, corrections[0].Amount.Equal(alpacadecimal.NewFromInt(-20)))

		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(20)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.EarningsSubAccountWithCostBasis(t, &zeroCostBasis)).Equal(alpacadecimal.Zero))
	})
}

func TestOnUsageBasedInvoiceUsageAccrued(t *testing.T) {
	t.Run("credit_then_invoice zero amount returns empty reference", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)

		ref, err := env.handler.OnInvoiceUsageAccrued(t.Context(), chargeusagebased.OnInvoiceUsageAccruedInput{
			Charge:        env.newCharge(productcatalog.CreditThenInvoiceSettlementMode),
			Run:           env.newRun(),
			ServicePeriod: timeutil.ClosedPeriod{From: env.Now().Add(-time.Hour), To: env.Now()},
			Amount:        alpacadecimal.Zero,
		})
		require.NoError(t, err)
		require.Empty(t, ref.TransactionGroupID)
	})
}

func TestOnUsageBasedPaymentAuthorized(t *testing.T) {
	t.Run("credit_then_invoice stages receivable funding from invoice usage", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)

		total := alpacadecimal.NewFromInt(40)
		_, err := env.handler.OnInvoiceUsageAccrued(t.Context(), chargeusagebased.OnInvoiceUsageAccruedInput{
			Charge:        env.newCharge(productcatalog.CreditThenInvoiceSettlementMode),
			Run:           env.newRunWithLine("line-1"),
			ServicePeriod: timeutil.ClosedPeriod{From: env.Now().Add(-time.Hour), To: env.Now()},
			Amount:        total,
		})
		require.NoError(t, err)

		charge := env.newCharge(productcatalog.CreditThenInvoiceSettlementMode)
		charge.Intent.InvoiceAt = env.Now().Add(-24 * time.Hour)
		eventTime := env.Now().Add(15 * time.Minute)
		clock.FreezeTime(eventTime)
		defer clock.UnFreeze()

		ref, err := env.handler.OnPaymentAuthorized(t.Context(), chargeusagebased.OnPaymentAuthorizedInput{
			Charge: charge,
			Run:    env.newRunWithInvoiceUsage("line-1", total),
		})
		require.NoError(t, err)
		require.NotEmpty(t, ref.TransactionGroupID)

		// Authorization only moves the receivable between status buckets.
		require.True(t, env.sumBalance(t, env.receivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-40)))
		require.True(t, env.sumBalance(t, env.washSubAccount(t)).Equal(alpacadecimal.Zero))
		// No revenue recognition happens here anymore.
		require.True(t, env.sumBalance(t, env.invoiceAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(40)))
		require.True(t, env.sumBalance(t, env.invoiceEarningsSubAccount(t)).Equal(alpacadecimal.Zero))

		for _, bookedAt := range env.transactionBookedAtTimes(t, ref.TransactionGroupID) {
			require.True(t, bookedAt.UTC().Equal(eventTime.UTC()))
			require.False(t, bookedAt.UTC().Equal(charge.Intent.InvoiceAt.UTC()))
		}
	})

	t.Run("zero invoice usage is a no-op", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)

		ref, err := env.handler.OnPaymentAuthorized(t.Context(), chargeusagebased.OnPaymentAuthorizedInput{
			Charge: env.newCharge(productcatalog.CreditThenInvoiceSettlementMode),
			Run:    env.newRunWithInvoiceUsage("line-1", alpacadecimal.Zero),
		})
		require.NoError(t, err)
		require.Empty(t, ref.TransactionGroupID)
	})
}

func TestOnUsageBasedPaymentSettled(t *testing.T) {
	t.Run("credit_then_invoice settles authorized receivable from wash", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)

		total := alpacadecimal.NewFromInt(25)
		_, err := env.handler.OnInvoiceUsageAccrued(t.Context(), chargeusagebased.OnInvoiceUsageAccruedInput{
			Charge:        env.newCharge(productcatalog.CreditThenInvoiceSettlementMode),
			Run:           env.newRunWithLine("line-1"),
			ServicePeriod: timeutil.ClosedPeriod{From: env.Now().Add(-time.Hour), To: env.Now()},
			Amount:        total,
		})
		require.NoError(t, err)

		authorizedCharge := env.newCharge(productcatalog.CreditThenInvoiceSettlementMode)
		authorizedCharge.Intent.InvoiceAt = env.Now().Add(-24 * time.Hour)
		_, err = env.handler.OnPaymentAuthorized(t.Context(), chargeusagebased.OnPaymentAuthorizedInput{
			Charge: authorizedCharge,
			Run:    env.newRunWithInvoiceUsage("line-1", total),
		})
		require.NoError(t, err)

		settledCharge := env.newCharge(productcatalog.CreditThenInvoiceSettlementMode)
		settledCharge.Intent.InvoiceAt = env.Now().Add(-48 * time.Hour)
		eventTime := env.Now().Add(30 * time.Minute)
		clock.FreezeTime(eventTime)
		defer clock.UnFreeze()

		ref, err := env.handler.OnPaymentSettled(t.Context(), chargeusagebased.OnPaymentSettledInput{
			Charge: settledCharge,
			Run:    env.newRunWithAuthorizedPayment("line-1", total),
		})
		require.NoError(t, err)
		require.NotEmpty(t, ref.TransactionGroupID)

		require.True(t, env.sumBalance(t, env.unknownReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.washSubAccount(t)).Equal(alpacadecimal.NewFromInt(-25)))

		for _, bookedAt := range env.transactionBookedAtTimes(t, ref.TransactionGroupID) {
			require.True(t, bookedAt.UTC().Equal(eventTime.UTC()))
			require.False(t, bookedAt.UTC().Equal(settledCharge.Intent.InvoiceAt.UTC()))
		}
	})

	t.Run("zero invoice usage is a no-op", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)

		ref, err := env.handler.OnPaymentSettled(t.Context(), chargeusagebased.OnPaymentSettledInput{
			Charge: env.newCharge(productcatalog.CreditThenInvoiceSettlementMode),
			Run:    env.newRunWithAuthorizedPaymentAndInvoiceUsage("line-1", alpacadecimal.NewFromInt(1), alpacadecimal.Zero),
		})
		require.NoError(t, err)
		require.Empty(t, ref.TransactionGroupID)
	})
}

type usageBasedHandlerTestEnv struct {
	*ledgertestutils.IntegrationEnv
	handler    chargeusagebased.Handler
	lineage    lineage.Service
	recognizer recognizer.Service
}

func newUsageBasedHandlerTestEnv(t *testing.T) *usageBasedHandlerTestEnv {
	base := ledgertestutils.NewIntegrationEnv(t, "chargeadapter-usagebased")
	collectorService := ledgercollector.NewService(ledgercollector.Config{
		Ledger: base.Deps.HistoricalLedger,
		Dependencies: transactions.ResolverDependencies{
			AccountService:    base.Deps.ResolversService,
			SubAccountService: base.Deps.AccountService,
		},
	})
	lineageAdapter, err := lineageadapter.New(lineageadapter.Config{
		Client: base.DB,
	})
	require.NoError(t, err)

	lineageService, err := lineageservice.New(lineageservice.Config{
		Adapter: lineageAdapter,
	})
	require.NoError(t, err)

	deps := transactions.ResolverDependencies{
		AccountService:    base.Deps.ResolversService,
		SubAccountService: base.Deps.AccountService,
	}
	recognizerService, err := recognizer.NewService(recognizer.Config{
		Ledger:             base.Deps.HistoricalLedger,
		Dependencies:       deps,
		Lineage:            lineageService,
		TransactionManager: enttx.NewCreator(base.DB),
	})
	require.NoError(t, err)

	return &usageBasedHandlerTestEnv{
		IntegrationEnv: base,
		handler: chargeadapter.NewUsageBasedHandler(base.Deps.HistoricalLedger, transactions.ResolverDependencies{
			AccountService:    base.Deps.ResolversService,
			SubAccountService: base.Deps.AccountService,
		}, collectorService),
		lineage:    lineageService,
		recognizer: recognizerService,
	}
}

func (e *usageBasedHandlerTestEnv) newCreditsOnlyCharge() chargeusagebased.Charge {
	return e.newCharge(productcatalog.CreditOnlySettlementMode)
}

func (e *usageBasedHandlerTestEnv) newCharge(settlementMode productcatalog.SettlementMode) chargeusagebased.Charge {
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
				ID: "usage-based-charge",
			},
			Intent: chargeusagebased.Intent{
				Intent: meta.Intent{
					Name:          "Usage based",
					ManagedBy:     billing.SystemManagedLine,
					CustomerID:    e.CustomerID.ID,
					Currency:      currencyx.Code("USD"),
					ServicePeriod: servicePeriod,
					BillingPeriod: servicePeriod,
				},
				InvoiceAt:      now,
				SettlementMode: settlementMode,
				FeatureKey:     "api_requests",
				Price:          *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
			},
			Status: chargeusagebased.StatusActiveFinalRealizationProcessing,
			State: chargeusagebased.State{
				FeatureID: featureID,
			},
		},
	}
}

func (e *usageBasedHandlerTestEnv) newRun() chargeusagebased.RealizationRun {
	now := time.Now().UTC()
	featureID := "feature-api-requests"

	return chargeusagebased.RealizationRun{
		RealizationRunBase: chargeusagebased.RealizationRunBase{
			ID: chargeusagebased.RealizationRunID(models.NamespacedID{
				Namespace: e.Namespace,
				ID:        "run-1",
			}),
			ManagedModel: models.ManagedModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			Type:            chargeusagebased.RealizationRunTypeFinalRealization,
			StoredAtLT:      now,
			ServicePeriodTo: now,
			MeteredQuantity: alpacadecimal.NewFromInt(30),
			FeatureID:       featureID,
			Totals: totals.Totals{
				Amount:       alpacadecimal.NewFromInt(30),
				CreditsTotal: alpacadecimal.NewFromInt(30),
				Total:        alpacadecimal.Zero,
			},
		},
	}
}

func (e *usageBasedHandlerTestEnv) newRunWithLine(lineID string) chargeusagebased.RealizationRun {
	run := e.newRun()
	run.LineID = &lineID
	return run
}

func (e *usageBasedHandlerTestEnv) newRunWithInvoiceUsage(lineID string, total alpacadecimal.Decimal) chargeusagebased.RealizationRun {
	run := e.newRunWithLine(lineID)
	run.InvoiceUsage = &invoicedusage.AccruedUsage{
		LineID:        &lineID,
		ServicePeriod: e.newCharge(productcatalog.CreditThenInvoiceSettlementMode).Intent.ServicePeriod,
		Mutable:       false,
		Totals: totals.Totals{
			Amount: total,
			Total:  total,
		},
	}

	return run
}

func (e *usageBasedHandlerTestEnv) newRunWithAuthorizedPayment(lineID string, total alpacadecimal.Decimal) chargeusagebased.RealizationRun {
	return e.newRunWithAuthorizedPaymentAndInvoiceUsage(lineID, total, total)
}

func (e *usageBasedHandlerTestEnv) newRunWithAuthorizedPaymentAndInvoiceUsage(lineID string, paymentAmount, invoiceUsageTotal alpacadecimal.Decimal) chargeusagebased.RealizationRun {
	run := e.newRunWithInvoiceUsage(lineID, invoiceUsageTotal)
	run.Payment = &payment.Invoiced{
		Payment: payment.Payment{
			NamespacedID: models.NamespacedID{
				Namespace: e.Namespace,
				ID:        "payment-1",
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: e.Now(),
				UpdatedAt: e.Now(),
			},
			Base: payment.Base{
				ServicePeriod: run.InvoiceUsage.ServicePeriod,
				Status:        payment.StatusAuthorized,
				Amount:        paymentAmount,
				Authorized: &ledgertransaction.TimedGroupReference{
					GroupReference: ledgertransaction.GroupReference{
						TransactionGroupID: "authorized-group",
					},
					Time: e.Now(),
				},
			},
		},
		LineID:    lineID,
		InvoiceID: "invoice-1",
	}

	return run
}

func (e *usageBasedHandlerTestEnv) fundPriority(t *testing.T, priority int, amount int64) ledger.SubAccount {
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

func (e *usageBasedHandlerTestEnv) creditAccruedSubAccount(t *testing.T) ledger.SubAccount {
	zeroCostBasis := alpacadecimal.Zero
	return e.AccruedSubAccountWithCostBasis(t, &zeroCostBasis)
}

func (e *usageBasedHandlerTestEnv) unknownAccruedSubAccount(t *testing.T) ledger.SubAccount {
	return e.AccruedSubAccountWithCostBasis(t, nil)
}

func (e *usageBasedHandlerTestEnv) unknownReceivableSubAccount(t *testing.T) ledger.SubAccount {
	return e.ReceivableSubAccountWithCostBasis(t, nil)
}

func (e *usageBasedHandlerTestEnv) authorizedReceivableSubAccount(t *testing.T) ledger.SubAccount {
	return e.ReceivableSubAccountWithCostBasisAndStatus(t, testInvoiceCostBasis(), ledger.TransactionAuthorizationStatusAuthorized)
}

func (e *usageBasedHandlerTestEnv) receivableSubAccount(t *testing.T) ledger.SubAccount {
	return e.ReceivableSubAccountWithCostBasis(t, testInvoiceCostBasis())
}

func (e *usageBasedHandlerTestEnv) washSubAccount(t *testing.T) ledger.SubAccount {
	return e.WashSubAccountWithCostBasis(t, testInvoiceCostBasis())
}

func (e *usageBasedHandlerTestEnv) invoiceAccruedSubAccount(t *testing.T) ledger.SubAccount {
	return e.AccruedSubAccountWithCostBasis(t, testInvoiceCostBasis())
}

func (e *usageBasedHandlerTestEnv) invoiceEarningsSubAccount(t *testing.T) ledger.SubAccount {
	return e.EarningsSubAccountWithCostBasis(t, testInvoiceCostBasis())
}

func (e *usageBasedHandlerTestEnv) unknownFboSubAccount(t *testing.T) ledger.SubAccount {
	subAccount, err := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       e.Currency,
		CreditPriority: ledger.DefaultCustomerFBOPriority,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *usageBasedHandlerTestEnv) sumBalance(t *testing.T, subAccount ledger.SubAccount) alpacadecimal.Decimal {
	return e.SumBalance(t, subAccount)
}

func (e *usageBasedHandlerTestEnv) transactionAnnotations(t *testing.T, groupID string) []models.Annotations {
	t.Helper()

	transactions, err := e.DB.LedgerTransaction.Query().
		Where(
			ledgertransactiondb.Namespace(e.Namespace),
			ledgertransactiondb.GroupID(groupID),
		).
		Order(
			ledgertransactiondb.ByCreatedAt(),
			ledgertransactiondb.ByID(),
		).
		All(t.Context())
	require.NoError(t, err)

	out := make([]models.Annotations, 0, len(transactions))
	for _, tx := range transactions {
		out = append(out, tx.Annotations)
	}

	return out
}

func (e *usageBasedHandlerTestEnv) transactionBookedAtTimes(t *testing.T, groupID string) []time.Time {
	t.Helper()

	transactions, err := e.DB.LedgerTransaction.Query().
		Where(
			ledgertransactiondb.Namespace(e.Namespace),
			ledgertransactiondb.GroupID(groupID),
		).
		Order(
			ledgertransactiondb.ByCreatedAt(),
			ledgertransactiondb.ByID(),
		).
		All(t.Context())
	require.NoError(t, err)

	out := make([]time.Time, 0, len(transactions))
	for _, tx := range transactions {
		out = append(out, tx.BookedAt)
	}

	return out
}

func (e *usageBasedHandlerTestEnv) recognizeCreditAccrued(t *testing.T, amount alpacadecimal.Decimal) string {
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

func (e *usageBasedHandlerTestEnv) createInitialLineages(t *testing.T, chargeID string, realizations creditrealization.Realizations) {
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

func (e *usageBasedHandlerTestEnv) activeSegmentsByRealization(t *testing.T, realizations creditrealization.Realizations) lineage.ActiveSegmentsByRealizationID {
	t.Helper()

	ids := make([]string, 0, len(realizations))
	for _, realization := range realizations {
		ids = append(ids, realization.ID)
	}

	segments, err := e.lineage.LoadActiveSegmentsByRealizationID(t.Context(), e.Namespace, ids)
	require.NoError(t, err)

	return segments
}

func (e *usageBasedHandlerTestEnv) assertRecognizedSegments(t *testing.T, realizations creditrealization.Realizations, recognitionGroupID string) lineage.ActiveSegmentsByRealizationID {
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

func (e *usageBasedHandlerTestEnv) ensureCharge(t *testing.T, chargeID string) {
	t.Helper()

	_, err := e.DB.Charge.Create().
		SetID(chargeID).
		SetNamespace(e.Namespace).
		SetType(meta.ChargeTypeUsageBased).
		Save(t.Context())
	require.NoError(t, err)
}

func (e *usageBasedHandlerTestEnv) realizationsFromAllocations(allocations creditrealization.CreateAllocationInputs) creditrealization.Realizations {
	now := time.Now().UTC()

	out := make(creditrealization.Realizations, 0, len(allocations))
	for i, allocation := range allocations.AsCreateInputs() {
		allocation.ID = ulid.Make().String()
		out = append(out, creditrealization.Realization{
			NamespacedModel: models.NamespacedModel{
				Namespace: e.Namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			CreateInput: allocation,
			SortHint:    i,
		})
	}

	return out
}
