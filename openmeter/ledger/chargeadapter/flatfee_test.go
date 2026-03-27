package chargeadapter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargeflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	ledgertransactiongroupdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransactiongroup"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
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

	t.Run("single bucket full coverage lands in accrued", func(t *testing.T) {
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

	t.Run("multiple buckets honor ascending priority", func(t *testing.T) {
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

	t.Run("insufficient balance returns partial realization", func(t *testing.T) {
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

	t.Run("zero amount returns no realization", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		priorityOne := env.fundPriority(t, 1, 100)

		realizations, err := env.handler.OnAssignedToInvoice(t.Context(), env.newAssignmentInput(alpacadecimal.NewFromInt(0)))
		require.NoError(t, err)
		require.Nil(t, realizations)
		require.True(t, env.sumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(100)))
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(0)))
	})

	t.Run("credit only advances uncovered amount", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		realizations, err := env.handler.OnAssignedToInvoice(t.Context(), env.newAssignmentInputWithMode(alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode))
		require.NoError(t, err)
		require.Len(t, realizations, 1)
		require.True(t, realizations[0].Amount.Equal(alpacadecimal.NewFromInt(30)))

		require.True(t, env.sumBalance(t, env.unknownReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-30)))
		require.True(t, env.sumBalance(t, env.unknownFboSubAccount(t)).Equal(alpacadecimal.Zero))
		require.True(t, env.sumBalance(t, env.unknownAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(30)))
	})
}

func TestOnFlatFeeStandardInvoiceUsageAccrued(t *testing.T) {
	t.Run("books receivable-backed usage into accrued", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		total := alpacadecimal.NewFromInt(50)
		ref, err := env.handler.OnInvoiceUsageAccrued(t.Context(), env.newAccrualInput(total))
		require.NoError(t, err)
		require.NotEmpty(t, ref.TransactionGroupID)

		require.True(t, env.sumBalance(t, env.receivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-50)))
		require.True(t, env.sumBalance(t, env.invoiceAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(50)))
	})

	t.Run("zero total returns empty reference", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		ref, err := env.handler.OnInvoiceUsageAccrued(t.Context(), env.newAccrualInput(alpacadecimal.NewFromInt(0)))
		require.NoError(t, err)
		require.Empty(t, ref.TransactionGroupID)
	})
}

func TestOnFlatFeePaymentAuthorized(t *testing.T) {
	t.Run("recognizes revenue from receivable-backed accrued", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		// First accrue usage: receivable → accrued
		total := alpacadecimal.NewFromInt(75)
		_, err := env.handler.OnInvoiceUsageAccrued(t.Context(), env.newAccrualInput(total))
		require.NoError(t, err)

		charge := env.newChargeWithAccruedUsage(total)
		ref, err := env.handler.OnPaymentAuthorized(t.Context(), charge)
		require.NoError(t, err)
		require.NotEmpty(t, ref.TransactionGroupID)

		// Receivable is only funded into the authorized staging bucket at authorization time.
		require.True(t, env.sumBalance(t, env.receivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-75)))
		require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(75)))
		require.True(t, env.sumBalance(t, env.washSubAccount(t)).Equal(alpacadecimal.NewFromInt(-75)))
		// Accrued drained, earnings recognized
		require.True(t, env.sumBalance(t, env.invoiceAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(0)))
		require.True(t, env.sumBalance(t, env.invoiceEarningsSubAccount(t)).Equal(alpacadecimal.NewFromInt(75)))
	})

	t.Run("recognizes revenue from mixed FBO and receivable", func(t *testing.T) {
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

		// Receivable funding stays staged until settlement.
		require.True(t, env.sumBalance(t, env.receivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-20)))
		require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(20)))
		// Accrued fully drained, all 60 recognized as earnings
		require.True(t, env.sumBalance(t, env.creditAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(0)))
		require.True(t, env.sumBalance(t, env.invoiceAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(0)))
		require.True(t, env.sumBalance(t, env.creditEarningsSubAccount(t)).Equal(alpacadecimal.NewFromInt(40)))
		require.True(t, env.sumBalance(t, env.invoiceEarningsSubAccount(t)).Equal(alpacadecimal.NewFromInt(20)))
	})

	t.Run("does not overdraw accrued during recognition", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		_, err := env.handler.OnInvoiceUsageAccrued(t.Context(), env.newAccrualInput(alpacadecimal.NewFromInt(30)))
		require.NoError(t, err)

		charge := env.newChargeWithAccruedUsage(alpacadecimal.NewFromInt(75))
		ref, err := env.handler.OnPaymentAuthorized(t.Context(), charge)
		require.NoError(t, err)
		require.NotEmpty(t, ref.TransactionGroupID)

		require.True(t, env.sumBalance(t, env.receivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-30)))
		require.True(t, env.sumBalance(t, env.authorizedReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(75)))
		require.True(t, env.sumBalance(t, env.washSubAccount(t)).Equal(alpacadecimal.NewFromInt(-75)))
		require.True(t, env.sumBalance(t, env.invoiceAccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(0)))
		require.True(t, env.sumBalance(t, env.invoiceEarningsSubAccount(t)).Equal(alpacadecimal.NewFromInt(30)))
	})
}

func TestOnFlatFeePaymentSettled(t *testing.T) {
	t.Run("settles authorized receivable into open receivable", func(t *testing.T) {
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

	t.Run("no receivable portion is a no-op", func(t *testing.T) {
		env := newFlatFeeHandlerTestEnv(t)

		ref, err := env.handler.OnPaymentSettled(t.Context(), env.newChargeWithCreditRealizationsAndAccruedUsage(nil, alpacadecimal.Zero))
		require.NoError(t, err)
		require.Empty(t, ref.TransactionGroupID)
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
	handler chargeflatfee.Handler
}

func newFlatFeeHandlerTestEnv(t *testing.T) *flatFeeHandlerTestEnv {
	base := ledgertestutils.NewIntegrationEnv(t, "chargeadapter-flatfee")

	return &flatFeeHandlerTestEnv{
		IntegrationEnv: base,
		handler: chargeadapter.NewFlatFeeHandler(
			base.Deps.HistoricalLedger,
			base.Deps.ResolversService,
			base.Deps.AccountService,
		),
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
			Status: meta.ChargeStatusActive,
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

func (e *flatFeeHandlerTestEnv) newBaseCharge(servicePeriod timeutil.ClosedPeriod, amount alpacadecimal.Decimal) chargeflatfee.Charge {
	return chargeflatfee.Charge{
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
		Status: meta.ChargeStatusActive,
	}
}

func (e *flatFeeHandlerTestEnv) newChargeWithAccruedUsage(total alpacadecimal.Decimal) chargeflatfee.Charge {
	now := time.Now().UTC()
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-time.Hour),
		To:   now,
	}

	charge := e.newBaseCharge(servicePeriod, total)
	charge.State.AccruedUsage = &invoicedusage.AccruedUsage{
		ServicePeriod: servicePeriod,
		Mutable:       true,
		Totals: totals.Totals{
			Amount: total,
			Total:  total,
		},
	}

	return charge
}

func (e *flatFeeHandlerTestEnv) newChargeWithCreditRealizationsAndAccruedUsage(realizations []creditrealization.CreateInput, accruedTotal alpacadecimal.Decimal) chargeflatfee.Charge {
	now := time.Now().UTC()
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-time.Hour),
		To:   now,
	}

	// Compute total amount (FBO + receivable portions)
	totalAmount := accruedTotal
	creditRealizations := make(creditrealization.Realizations, 0, len(realizations))
	for i, r := range realizations {
		totalAmount = totalAmount.Add(r.Amount)
		creditRealizations = append(creditRealizations, creditrealization.Realization{
			NamespacedID: models.NamespacedID{
				Namespace: e.Namespace,
				ID:        fmt.Sprintf("cr-%d", i),
			},
			CreateInput: r,
		})
	}

	charge := e.newBaseCharge(servicePeriod, totalAmount)
	charge.State.CreditRealizations = creditRealizations
	charge.State.AccruedUsage = &invoicedusage.AccruedUsage{
		ServicePeriod: servicePeriod,
		Mutable:       true,
		Totals: totals.Totals{
			Amount: accruedTotal,
			Total:  accruedTotal,
		},
	}

	return charge
}

func (e *flatFeeHandlerTestEnv) newChargeWithPayment(total alpacadecimal.Decimal, authRef ledgertransaction.GroupReference) chargeflatfee.Charge {
	now := time.Now().UTC()
	charge := e.newChargeWithAccruedUsage(total)
	charge.State.Payment = &payment.Invoiced{
		Payment: payment.Payment{
			NamespacedID: models.NamespacedID{
				Namespace: e.Namespace,
				ID:        "payment-1",
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			Base: payment.Base{
				ServicePeriod: charge.Intent.ServicePeriod,
				Amount:        total,
				Authorized: &ledgertransaction.TimedGroupReference{
					GroupReference: authRef,
					Time:           now,
				},
				Status: payment.StatusAuthorized,
			},
		},
		LineID: "line-1",
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

func (e *flatFeeHandlerTestEnv) brokerageSubAccount(t *testing.T) ledger.SubAccount {
	return e.BrokerageSubAccount(t)
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
