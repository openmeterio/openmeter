package collector

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type accrualCollector struct {
	ledger ledger.Ledger
	deps   transactions.ResolverDependencies
}

type collectedInputs []ledger.TransactionInput

func (c *accrualCollector) collect(ctx context.Context, input CollectToAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	if input.Amount.IsZero() {
		return nil, nil
	}

	inputs, err := c.resolveCollectedInputs(ctx, input, input.Amount)
	if err != nil {
		return nil, err
	}

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

	return collectedInputs(inputs).toCreditRealizations(input.ServicePeriod, transactionGroup.ID().ID), nil
}

func (c *accrualCollector) resolveCollectedInputs(ctx context.Context, input CollectToAccruedInput, amount alpacadecimal.Decimal) ([]ledger.TransactionInput, error) {
	inputs, err := transactions.ResolveTransactions(
		ctx,
		c.deps,
		c.resolutionScope(input),
		transactions.TransferCustomerFBOToAccruedTemplate{
			At:       input.At,
			Amount:   amount,
			Currency: input.Currency,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("resolve transactions: %w", err)
	}

	return inputs, nil
}

func (c *accrualCollector) resolveAdvanceInputs(ctx context.Context, input CollectToAccruedInput, amount alpacadecimal.Decimal) ([]ledger.TransactionInput, error) {
	inputs, err := transactions.ResolveTransactions(
		ctx,
		c.deps,
		c.resolutionScope(input),
		transactions.IssueCustomerReceivableTemplate{
			At:       input.At,
			Amount:   amount,
			Currency: input.Currency,
		},
		transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
			At:       input.At,
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
		CustomerID: customer.CustomerID{
			Namespace: input.Namespace,
			ID:        input.CustomerID,
		},
		Namespace: input.Namespace,
	}
}

func (i collectedInputs) toCreditRealizations(servicePeriod timeutil.ClosedPeriod, transactionGroupID string) creditrealization.CreateAllocationInputs {
	out := make(creditrealization.CreateAllocationInputs, 0, len(i))
	for _, input := range i {
		if input == nil {
			continue
		}

		annotations := creditRealizationAnnotationsForCollectedInput(input)
		// One realization row per FBO debit on the resolved inputs (the spend from balance).
		for _, entry := range input.EntryInputs() {
			if entry.Amount().IsNegative() && entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO {
				out = append(out, creditrealization.CreateAllocationInput{
					Annotations:   annotations,
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
	templateName, err := ledger.TransactionTemplateNameFromAnnotations(input.Annotations())
	if err != nil {
		return input.Annotations()
	}

	var originKind creditrealization.LineageOriginKind
	switch templateName {
	case transactions.TemplateName(transactions.TransferCustomerFBOToAccruedTemplate{}):
		originKind = creditrealization.LineageOriginKindRealCredit
	case transactions.TemplateName(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}):
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
