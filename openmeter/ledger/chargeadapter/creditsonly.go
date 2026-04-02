package chargeadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type creditsOnlyAccrualRequest struct {
	Namespace      string
	ChargeID       string
	CustomerID     string
	At             time.Time
	Currency       currencyx.Code
	SettlementMode productcatalog.SettlementMode
}

func allocateCreditsToAccrued(
	ctx context.Context,
	ledgerService ledger.Ledger,
	deps transactions.ResolverDependencies,
	req creditsOnlyAccrualRequest,
	amount alpacadecimal.Decimal,
) (string, []ledger.TransactionInput, error) {
	customerID := customer.CustomerID{
		Namespace: req.Namespace,
		ID:        req.CustomerID,
	}
	annotations := ledger.ChargeAnnotations(models.NamespacedID{
		Namespace: req.Namespace,
		ID:        req.ChargeID,
	})

	inputs, err := transactions.ResolveTransactions(
		ctx,
		deps,
		transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  req.Namespace,
		},
		transactions.TransferCustomerFBOToAccruedTemplate{
			At:       req.At,
			Amount:   amount,
			Currency: req.Currency,
		},
	)
	if err != nil {
		return "", nil, fmt.Errorf("resolve transactions: %w", err)
	}

	collectedAmount := sumCollectedFBOAmount(inputs...)
	shortfall := amount.Sub(collectedAmount)
	if req.SettlementMode == productcatalog.CreditOnlySettlementMode && shortfall.IsPositive() {
		advanceInputs, err := transactions.ResolveTransactions(
			ctx,
			deps,
			transactions.ResolutionScope{
				CustomerID: customerID,
				Namespace:  req.Namespace,
			},
			transactions.IssueCustomerReceivableTemplate{
				At:       req.At,
				Amount:   shortfall,
				Currency: req.Currency,
			},
			transactions.TransferCustomerFBOBucketToAccruedTemplate{
				At:       req.At,
				Amount:   shortfall,
				Currency: req.Currency,
			},
		)
		if err != nil {
			return "", nil, fmt.Errorf("resolve advance transactions: %w", err)
		}

		inputs = append(inputs, advanceInputs...)
	}

	if len(inputs) == 0 {
		return "", nil, nil
	}

	transactionGroup, err := ledgerService.CommitGroup(ctx, transactions.GroupInputs(
		req.Namespace,
		annotations,
		inputs...,
	))
	if err != nil {
		return "", nil, fmt.Errorf("commit ledger transaction group: %w", err)
	}

	return transactionGroup.ID().ID, inputs, nil
}

func creditRealizationsFromCollectedInputs(servicePeriod timeutil.ClosedPeriod, transactionGroupID string, inputs ...ledger.TransactionInput) creditrealization.CreateAllocationInputs {
	out := make(creditrealization.CreateAllocationInputs, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		for _, entry := range input.EntryInputs() {
			if entry.Amount().IsNegative() && entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO {
				out = append(out, creditrealization.CreateAllocationInput{
					ServicePeriod: servicePeriod,
					Amount:        entry.Amount().Abs(),
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: transactionGroupID,
					},
				})
			}
		}
	}

	return out
}

func sumCollectedFBOAmount(inputs ...ledger.TransactionInput) alpacadecimal.Decimal {
	total := alpacadecimal.Zero
	for _, input := range inputs {
		if input == nil {
			continue
		}
		for _, entry := range input.EntryInputs() {
			if entry.Amount().IsNegative() && entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO {
				total = total.Add(entry.Amount().Abs())
			}
		}
	}

	return total
}

func settledBalanceForSubAccount(ctx context.Context, subAccount ledger.SubAccount) (alpacadecimal.Decimal, error) {
	balance, err := subAccount.GetBalance(ctx)
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get balance for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return balance.Settled(), nil
}
