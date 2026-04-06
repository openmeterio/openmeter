package chargeadapter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	chargeusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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
}

type usageBasedHandlerTestEnv struct {
	*ledgertestutils.IntegrationEnv
	handler chargeusagebased.Handler
}

func newUsageBasedHandlerTestEnv(t *testing.T) *usageBasedHandlerTestEnv {
	base := ledgertestutils.NewIntegrationEnv(t, "chargeadapter-usagebased")

	return &usageBasedHandlerTestEnv{
		IntegrationEnv: base,
		handler: chargeadapter.NewUsageBasedHandler(
			base.Deps.HistoricalLedger,
			base.Deps.ResolversService,
			base.Deps.AccountService,
		),
	}
}

func (e *usageBasedHandlerTestEnv) newCreditsOnlyCharge() chargeusagebased.Charge {
	now := time.Now().UTC()
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
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				FeatureKey:     "api_requests",
				Price:          *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
			},
			Status: chargeusagebased.StatusActiveFinalRealizationProcessing,
			State:  chargeusagebased.State{},
		},
	}
}

func (e *usageBasedHandlerTestEnv) newRun() chargeusagebased.RealizationRun {
	now := time.Now().UTC()

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
			Type:          chargeusagebased.RealizationRunTypeFinalRealization,
			AsOf:          now,
			CollectionEnd: now,
			MeterValue:    alpacadecimal.NewFromInt(30),
			Totals: totals.Totals{
				Amount:       alpacadecimal.NewFromInt(30),
				CreditsTotal: alpacadecimal.NewFromInt(30),
				Total:        alpacadecimal.Zero,
			},
		},
	}
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
		transactions.FundCustomerReceivableTemplate{
			At:        e.Now(),
			Amount:    alpacadecimal.NewFromInt(amount),
			Currency:  e.Currency,
			CostBasis: &costBasis,
		},
		transactions.SettleCustomerReceivablePaymentTemplate{
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

func (e *usageBasedHandlerTestEnv) realizationsFromAllocations(allocations creditrealization.CreateAllocationInputs) creditrealization.Realizations {
	now := time.Now().UTC()

	out := make(creditrealization.Realizations, 0, len(allocations))
	for i, allocation := range allocations.AsCreateInputs() {
		allocation.ID = fmt.Sprintf("cr-%d", i)
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
