package collector

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type accrualCollector struct {
	ledger             ledger.Ledger
	deps               transactions.ResolverDependencies
	breakage           breakage.Service
	transactionManager transaction.Creator
}

type collectedInputs []ledger.TransactionInput

type resolvedCollectedInputs struct {
	inputs          []ledger.TransactionInput
	breakagePending []breakage.PendingRecord
}

func (c *accrualCollector) collect(ctx context.Context, input CollectToAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	run := func(ctx context.Context) (creditrealization.CreateAllocationInputs, error) {
		if input.Amount.IsZero() {
			return nil, nil
		}

		resolved, err := c.resolveCollectedInputs(ctx, input, input.Amount)
		if err != nil {
			return nil, err
		}
		inputs := resolved.inputs

		// Credit-only: if the wallet didn't cover the full accrual, issue advance and
		// move that slice through the advance-to-accrued path.
		if shortfall := input.Amount.Sub(collectedInputs(inputs).collectedFBOAmount()); c.shouldAdvanceShortfall(input, shortfall) {
			advanceInputs, err := c.resolveAdvanceInputs(ctx, input, shortfall)
			if err != nil {
				return nil, err
			}

			inputs = append(inputs, advanceInputs...)
		}

		if len(inputs) == 0 {
			return nil, nil
		}

		groupAnnotations := input.Annotations
		if groupAnnotations == nil {
			groupAnnotations = ledger.ChargeAnnotations(models.NamespacedID{
				Namespace: input.Namespace,
				ID:        input.ChargeID,
			})
		}

		for i, txInput := range inputs {
			if txInput != nil {
				inputs[i] = transactions.WithAnnotations(txInput, groupAnnotations)
			}
		}

		transactionGroup, err := c.ledger.CommitGroup(ctx, transactions.GroupInputs(
			input.Namespace,
			groupAnnotations,
			inputs...,
		))
		if err != nil {
			return nil, fmt.Errorf("commit ledger transaction group: %w", err)
		}

		if c.breakage != nil {
			// Breakage rows describe committed breakage ledger transactions, so
			// they must be persisted in the same transaction context as the
			// ledger group.
			if err := c.breakage.PersistCommittedRecords(ctx, resolved.breakagePending, transactionGroup); err != nil {
				return nil, fmt.Errorf("persist breakage records: %w", err)
			}
		}

		return collectedInputs(inputs).toCreditRealizations(input.ServicePeriod, transactionGroup.ID().ID), nil
	}

	return transaction.Run(ctx, c.transactionManager, run)
}

func (c *accrualCollector) resolveCollectedInputs(ctx context.Context, input CollectToAccruedInput, amount alpacadecimal.Decimal) (resolvedCollectedInputs, error) {
	if err := ledger.ValidateTransactionAmount(amount); err != nil {
		return resolvedCollectedInputs{}, fmt.Errorf("amount: %w", err)
	}

	if err := ledger.ValidateCurrency(input.Currency); err != nil {
		return resolvedCollectedInputs{}, fmt.Errorf("currency: %w", err)
	}

	selections, err := c.collectCustomerFBOSelections(ctx, c.customerID(input), input.Currency, amount, input.SourceBalanceAsOf)
	if err != nil {
		return resolvedCollectedInputs{}, fmt.Errorf("collect customer FBO: %w", err)
	}

	if len(selections) == 0 {
		return resolvedCollectedInputs{}, nil
	}

	sources := fboCollectionSelections(selections).postingAmounts()
	inputs, err := transactions.ResolveTransactions(
		ctx,
		c.deps,
		c.resolutionScope(input),
		transactions.TransferCustomerFBOToAccruedTemplate{
			At:       input.BookedAt,
			Currency: input.Currency,
			Sources:  sources,
		},
	)
	if err != nil {
		return resolvedCollectedInputs{}, fmt.Errorf("resolve transactions: %w", err)
	}

	var pending []breakage.PendingRecord
	if c.breakage != nil {
		for idx, selection := range selections {
			if selection.source.breakagePlan == nil {
				continue
			}

			releaseInput, releaseRecord, err := c.breakage.ReleasePlan(ctx, breakage.ReleasePlanInput{
				Plan:                   *selection.source.breakagePlan,
				Amount:                 selection.amount,
				SourceKind:             breakage.SourceKindUsage,
				SourceEntryIdentityKey: transactions.NewCollectionSourceIdentityKey(idx),
			})
			if err != nil {
				return resolvedCollectedInputs{}, fmt.Errorf("resolve breakage release: %w", err)
			}

			inputs = append(inputs, releaseInput)
			pending = append(pending, releaseRecord)
		}
	}

	return resolvedCollectedInputs{
		inputs:          inputs,
		breakagePending: pending,
	}, nil
}

func (c *accrualCollector) resolveAdvanceInputs(ctx context.Context, input CollectToAccruedInput, amount alpacadecimal.Decimal) ([]ledger.TransactionInput, error) {
	inputs, err := transactions.ResolveTransactions(
		ctx,
		c.deps,
		c.resolutionScope(input),
		transactions.IssueCustomerReceivableTemplate{
			At:       input.BookedAt,
			Amount:   amount,
			Currency: input.Currency,
		},
		transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
			At:       input.BookedAt,
			Amount:   amount,
			Currency: input.Currency,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("resolve advance transactions: %w", err)
	}

	return inputs, nil
}

func (c *accrualCollector) shouldAdvanceShortfall(input CollectToAccruedInput, shortfall alpacadecimal.Decimal) bool {
	return input.SettlementMode == productcatalog.CreditOnlySettlementMode && shortfall.IsPositive()
}

func (c *accrualCollector) resolutionScope(input CollectToAccruedInput) transactions.ResolutionScope {
	return transactions.ResolutionScope{
		CustomerID: c.customerID(input),
		Namespace:  input.Namespace,
	}
}

func (c *accrualCollector) customerID(input CollectToAccruedInput) customer.CustomerID {
	return customer.CustomerID{
		Namespace: input.Namespace,
		ID:        input.CustomerID,
	}
}

func (i collectedInputs) toCreditRealizations(servicePeriod timeutil.ClosedPeriod, transactionGroupID string) creditrealization.CreateAllocationInputs {
	out := make(creditrealization.CreateAllocationInputs, 0, len(i))
	for _, input := range i {
		if input == nil {
			continue
		}

		annotations := creditRealizationAnnotationsForCollectedInput(input)
		// Keep billing realization granularity at the FBO sub-account bucket.
		// Entry identity may split same-sub-account collection internally, but
		// that should not leak as separate credit realizations.
		amountsBySubAccountID := make(map[string]alpacadecimal.Decimal)
		subAccountOrder := make([]string, 0)
		for _, entry := range input.EntryInputs() {
			if entry.Amount().IsNegative() && entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO {
				subAccountID := entry.PostingAddress().SubAccountID()
				if _, ok := amountsBySubAccountID[subAccountID]; !ok {
					subAccountOrder = append(subAccountOrder, subAccountID)
				}
				amountsBySubAccountID[subAccountID] = amountsBySubAccountID[subAccountID].Add(entry.Amount().Abs())
			}
		}

		for _, subAccountID := range subAccountOrder {
			amount := amountsBySubAccountID[subAccountID]
			if !amount.IsPositive() {
				continue
			}

			out = append(out, creditrealization.CreateAllocationInput{
				Annotations:   annotations,
				ServicePeriod: servicePeriod,
				Amount:        amount,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: transactionGroupID,
				},
			})
		}
	}

	return out
}

func (i collectedInputs) collectedFBOAmount() alpacadecimal.Decimal {
	total := alpacadecimal.Zero
	for _, input := range i {
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

func creditRealizationAnnotationsForCollectedInput(input ledger.TransactionInput) models.Annotations {
	templateCode, err := ledger.TransactionTemplateCodeFromAnnotations(input.Annotations())
	if err != nil {
		return input.Annotations()
	}

	var originKind creditrealization.LineageOriginKind
	switch templateCode {
	case transactions.TemplateCode(transactions.TransferCustomerFBOToAccruedTemplate{}):
		originKind = creditrealization.LineageOriginKindRealCredit
	case transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}):
		originKind = creditrealization.LineageOriginKindAdvance
	default:
		return input.Annotations()
	}

	annotations, err := input.Annotations().Merge(creditrealization.LineageAnnotations(originKind))
	if err != nil {
		return input.Annotations()
	}

	return annotations
}
