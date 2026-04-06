package chargeadapter_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargecreditpurchase "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	ledgertransactiongroupdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransactiongroup"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestOnPromotionalCreditPurchase(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)

	charge := env.newPromotionalCharge(alpacadecimal.NewFromInt(100))
	ref, err := env.handler.OnPromotionalCreditPurchase(t.Context(), charge)
	require.NoError(t, err)
	require.NotEmpty(t, ref.TransactionGroupID)
	require.Equal(
		t,
		ledger.ChargeAnnotations(models.NamespacedID{Namespace: env.Namespace, ID: charge.ID}),
		env.transactionGroupAnnotations(t, ref.TransactionGroupID),
	)

	require.True(t, env.sumBalance(t, env.fboSubAccount(t, alpacadecimal.Zero)).Equal(alpacadecimal.NewFromInt(100)))
	require.True(t, env.sumBalance(t, env.receivableSubAccount(t, alpacadecimal.Zero)).Equal(alpacadecimal.NewFromInt(-100)))
}

func TestOnCreditPurchaseInitiated(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)

	costBasis := mustDecimal(t, "0.5")
	charge := env.newExternalCharge(alpacadecimal.NewFromInt(100), costBasis)
	ref, err := env.handler.OnCreditPurchaseInitiated(t.Context(), charge)
	require.NoError(t, err)
	require.NotEmpty(t, ref.TransactionGroupID)

	require.True(t, env.sumBalance(t, env.fboSubAccount(t, costBasis)).Equal(alpacadecimal.NewFromInt(100)))
	require.True(t, env.sumBalance(t, env.receivableSubAccount(t, costBasis)).Equal(alpacadecimal.NewFromInt(-100)))
}

func TestOnCreditPurchaseInitiated_OnlyIssuesExcessBeyondAdvance(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)
	env.createAdvanceExposure(t, alpacadecimal.NewFromInt(40))

	costBasis := mustDecimal(t, "0.5")
	charge := env.newExternalCharge(alpacadecimal.NewFromInt(100), costBasis)
	ref, err := env.handler.OnCreditPurchaseInitiated(t.Context(), charge)
	require.NoError(t, err)
	require.NotEmpty(t, ref.TransactionGroupID)

	require.True(t, env.sumBalance(t, env.fboSubAccount(t, costBasis)).Equal(alpacadecimal.NewFromInt(60)))
	require.True(t, env.sumBalance(t, env.receivableSubAccount(t, costBasis)).Equal(alpacadecimal.NewFromInt(-100)))
	require.True(t, env.sumBalance(t, env.unknownReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.unknownAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.accruedSubAccount(t, costBasis)).Equal(alpacadecimal.NewFromInt(40)))
}

func TestOnCreditPurchasePaymentAuthorized(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)

	costBasis := mustDecimal(t, "0.5")
	charge := env.newExternalCharge(alpacadecimal.NewFromInt(100), costBasis)

	_, err := env.handler.OnCreditPurchaseInitiated(t.Context(), charge)
	require.NoError(t, err)

	ref, err := env.handler.OnCreditPurchasePaymentAuthorized(t.Context(), charge)
	require.NoError(t, err)
	require.NotEmpty(t, ref.TransactionGroupID)

	require.True(t, env.sumBalance(t, env.receivableSubAccount(t, costBasis)).Equal(alpacadecimal.NewFromInt(-100)))
	require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t, costBasis)).Equal(alpacadecimal.NewFromInt(100)))
	require.True(t, env.sumBalance(t, env.washSubAccount(t, costBasis)).Equal(alpacadecimal.NewFromInt(-100)))
	require.True(t, env.sumBalance(t, env.fboSubAccount(t, costBasis)).Equal(alpacadecimal.NewFromInt(100)))
}

func TestOnCreditPurchasePaymentSettled(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)

	costBasis := mustDecimal(t, "0.5")
	charge := env.newExternalCharge(alpacadecimal.NewFromInt(100), costBasis)
	_, err := env.handler.OnCreditPurchaseInitiated(t.Context(), charge)
	require.NoError(t, err)

	_, err = env.handler.OnCreditPurchasePaymentAuthorized(t.Context(), charge)
	require.NoError(t, err)

	ref, err := env.handler.OnCreditPurchasePaymentSettled(t.Context(), charge)
	require.NoError(t, err)
	require.NotEmpty(t, ref.TransactionGroupID)

	require.True(t, env.sumBalance(t, env.receivableSubAccount(t, costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t, costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.washSubAccount(t, costBasis)).Equal(alpacadecimal.NewFromInt(-100)))
	require.True(t, env.sumBalance(t, env.fboSubAccount(t, costBasis)).Equal(alpacadecimal.NewFromInt(100)))
}

func TestOnCreditPurchasePaymentSettled_BacksAdvanceBeforeTopUp(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)
	env.createAdvanceExposure(t, alpacadecimal.NewFromInt(40))

	costBasis := mustDecimal(t, "0.5")
	charge := env.newExternalCharge(alpacadecimal.NewFromInt(100), costBasis)

	_, err := env.handler.OnCreditPurchaseInitiated(t.Context(), charge)
	require.NoError(t, err)

	_, err = env.handler.OnCreditPurchasePaymentAuthorized(t.Context(), charge)
	require.NoError(t, err)

	ref, err := env.handler.OnCreditPurchasePaymentSettled(t.Context(), charge)
	require.NoError(t, err)
	require.NotEmpty(t, ref.TransactionGroupID)

	require.True(t, env.sumBalance(t, env.receivableSubAccount(t, costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t, costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.unknownReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.unknownAccruedSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.accruedSubAccount(t, costBasis)).Equal(alpacadecimal.NewFromInt(40)))
	require.True(t, env.sumBalance(t, env.fboSubAccount(t, costBasis)).Equal(alpacadecimal.NewFromInt(60)))
}

type creditPurchaseHandlerTestEnv struct {
	*ledgertestutils.IntegrationEnv
	handler chargecreditpurchase.Handler
}

func newCreditPurchaseHandlerTestEnv(t *testing.T) *creditPurchaseHandlerTestEnv {
	base := ledgertestutils.NewIntegrationEnv(t, "chargeadapter-creditpurchase")

	return &creditPurchaseHandlerTestEnv{
		IntegrationEnv: base,
		handler: chargeadapter.NewCreditPurchaseHandler(
			base.Deps.HistoricalLedger,
			base.Deps.ResolversService,
			base.Deps.AccountService,
		),
	}
}

func (e *creditPurchaseHandlerTestEnv) newPromotionalCharge(amount alpacadecimal.Decimal) chargecreditpurchase.Charge {
	now := time.Now().UTC()
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-time.Hour),
		To:   now,
	}

	return chargecreditpurchase.Charge{
		ManagedResource: meta.ManagedResource{
			NamespacedModel: models.NamespacedModel{
				Namespace: e.Namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			ID: "credit-purchase-charge",
		},
		Intent: chargecreditpurchase.Intent{
			Intent: meta.Intent{
				Name:              "Promotional Credit Purchase",
				ManagedBy:         billing.SystemManagedLine,
				CustomerID:        e.CustomerID.ID,
				Currency:          currencyx.Code("USD"),
				ServicePeriod:     servicePeriod,
				FullServicePeriod: servicePeriod,
				BillingPeriod:     servicePeriod,
			},
			CreditAmount: amount,
			Settlement:   chargecreditpurchase.NewSettlement(chargecreditpurchase.PromotionalSettlement{}),
		},
		Status: meta.ChargeStatusCreated,
	}
}

func (e *creditPurchaseHandlerTestEnv) newExternalCharge(amount, costBasis alpacadecimal.Decimal) chargecreditpurchase.Charge {
	now := time.Now().UTC()
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-time.Hour),
		To:   now,
	}

	return chargecreditpurchase.Charge{
		ManagedResource: meta.ManagedResource{
			NamespacedModel: models.NamespacedModel{
				Namespace: e.Namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			ID: "credit-purchase-charge",
		},
		Intent: chargecreditpurchase.Intent{
			Intent: meta.Intent{
				Name:              "External Credit Purchase",
				ManagedBy:         billing.SystemManagedLine,
				CustomerID:        e.CustomerID.ID,
				Currency:          currencyx.Code("USD"),
				ServicePeriod:     servicePeriod,
				FullServicePeriod: servicePeriod,
				BillingPeriod:     servicePeriod,
			},
			CreditAmount: amount,
			Settlement: chargecreditpurchase.NewSettlement(chargecreditpurchase.ExternalSettlement{
				InitialStatus: chargecreditpurchase.CreatedInitialPaymentSettlementStatus,
				GenericSettlement: chargecreditpurchase.GenericSettlement{
					Currency:  currencyx.Code("USD"),
					CostBasis: costBasis,
				},
			}),
		},
		Status: meta.ChargeStatusCreated,
	}
}

func (e *creditPurchaseHandlerTestEnv) fboSubAccount(t *testing.T, costBasis alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       e.Currency,
		CostBasis:      &costBasis,
		CreditPriority: ledger.DefaultCustomerFBOPriority,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) unknownReceivableSubAccount(t *testing.T) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       e.Currency,
		CostBasis:                      nil,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) unknownAccruedSubAccount(t *testing.T) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:  e.Currency,
		CostBasis: nil,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) accruedSubAccount(t *testing.T, costBasis alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:  e.Currency,
		CostBasis: &costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) receivableSubAccount(t *testing.T, costBasis alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       e.Currency,
		CostBasis:                      &costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) authorizedReceivableSubAccount(t *testing.T, costBasis alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       e.Currency,
		CostBasis:                      &costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusAuthorized,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) washSubAccount(t *testing.T, costBasis alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.BusinessAccounts.WashAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:  e.Currency,
		CostBasis: &costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) sumBalance(t *testing.T, subAccount ledger.SubAccount) alpacadecimal.Decimal {
	return e.SumBalance(t, subAccount)
}

func (e *creditPurchaseHandlerTestEnv) createAdvanceExposure(t *testing.T, amount alpacadecimal.Decimal) {
	t.Helper()

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
			At:       e.Now(),
			Amount:   amount,
			Currency: e.Currency,
		},
		transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
			At:       e.Now(),
			Amount:   amount,
			Currency: e.Currency,
		},
	)
	require.NoError(t, err)

	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}

func (e *creditPurchaseHandlerTestEnv) transactionGroupAnnotations(t *testing.T, groupID string) models.Annotations {
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

func mustDecimal(t *testing.T, raw string) alpacadecimal.Decimal {
	t.Helper()

	value, err := alpacadecimal.NewFromString(raw)
	require.NoError(t, err)

	return value
}
