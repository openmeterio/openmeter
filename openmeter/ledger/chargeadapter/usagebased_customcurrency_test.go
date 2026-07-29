package chargeadapter_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargeusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/currency"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

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
		customCurrencyIdentity := &ledger.CustomCurrencyIdentity{ID: customCurrencyValue.ID, Precision: 2}
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

		require.True(t, env.sumBalance(t, env.customFBOSubAccountForUsageBased(t, customCurrency, customCurrencyIdentity, &settlementCurrency, costBasis)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.customAccruedSubAccountForUsageBased(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis)).Equal(alpacadecimal.NewFromInt(30)))
	})

	t.Run("credit_only advances an uncovered custom currency amount", func(t *testing.T) {
		env := newUsageBasedHandlerTestEnv(t)
		customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
		customCurrency := customCurrencyValue.GetCode()
		customCurrencyIdentity := &ledger.CustomCurrencyIdentity{ID: customCurrencyValue.ID, Precision: 2}
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
	customCurrencyIdentity := &ledger.CustomCurrencyIdentity{ID: customCurrencyValue.ID, Precision: 2}
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

// fundCustomFBO books a known-cost-basis, fiat-sourced custom currency FBO
// balance the way a credit purchase issues one: crediting FBO and debiting an
// open receivable in the same posting. The receivable side (and its later
// fiat conversion/settlement) is irrelevant to credit_only collection, which
// only reads FBO/accrued balances, so it is left open here.
func (e *usageBasedHandlerTestEnv) fundCustomFBO(t *testing.T, currency currencyx.Code, customCurrency *ledger.CustomCurrencyIdentity, costBasisCurrency *currencyx.Code, costBasis alpacadecimal.Decimal, amount alpacadecimal.Decimal) {
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
			Currency:          currency,
			CustomCurrency:    customCurrency,
			CostBasisCurrency: costBasisCurrency,
			CostBasis:         &costBasis,
		},
	)
	require.NoError(t, err)

	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}

func (e *usageBasedHandlerTestEnv) customFBOSubAccountForUsageBased(t *testing.T, currency currencyx.Code, customCurrency *ledger.CustomCurrencyIdentity, costBasisCurrency *currencyx.Code, costBasis alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:          currency,
		CustomCurrency:    customCurrency,
		CostBasisCurrency: costBasisCurrency,
		CostBasis:         &costBasis,
		CreditPriority:    ledger.DefaultCustomerFBOPriority,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *usageBasedHandlerTestEnv) customAccruedSubAccountForUsageBased(t *testing.T, currency currencyx.Code, customCurrency *ledger.CustomCurrencyIdentity, costBasisCurrency *currencyx.Code, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:          currency,
		CustomCurrency:    customCurrency,
		CostBasisCurrency: costBasisCurrency,
		TaxCode:           lo.ToPtr(testChargeTaxCodeID),
		CostBasis:         costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *usageBasedHandlerTestEnv) customUnknownAccruedSubAccountForUsageBased(t *testing.T, currency currencyx.Code, customCurrency *ledger.CustomCurrencyIdentity) ledger.SubAccount {
	t.Helper()

	return e.customAccruedSubAccountForUsageBased(t, currency, customCurrency, nil, nil)
}

func (e *usageBasedHandlerTestEnv) customUnknownFboSubAccountForUsageBased(t *testing.T, currency currencyx.Code, customCurrency *ledger.CustomCurrencyIdentity) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       currency,
		CustomCurrency: customCurrency,
		CreditPriority: ledger.DefaultCustomerFBOPriority,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *usageBasedHandlerTestEnv) customReceivableSubAccountForUsageBasedFeature(t *testing.T, currency currencyx.Code, customCurrency *ledger.CustomCurrencyIdentity, featureKey string) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       currency,
		CustomCurrency:                 customCurrency,
		CostBasis:                      nil,
		Features:                       []string{featureKey},
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	require.NoError(t, err)

	return subAccount
}
