package collector

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type accrualCollector struct {
	ledger             ledger.Ledger
	advance            advance.Service
	deps               transactions.ResolverDependencies
	breakage           breakage.Service
	accountLocker      ledger.AccountLocker
	transactionManager transaction.Creator
}

type collectedInputs []ledger.TransactionInput

func (c *accrualCollector) collectToAccrued(ctx context.Context, input CollectToAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	run := func(ctx context.Context) (creditrealization.CreateAllocationInputs, error) {
		if input.Amount.IsZero() {
			return nil, nil
		}

		groupAnnotations := input.Annotations
		if groupAnnotations == nil {
			groupAnnotations = ledger.ChargeAnnotations(models.NamespacedID{
				Namespace: input.Namespace,
				ID:        input.ChargeID,
			})
		}

		posting := breakage.PostingInput{
			TransactionGroupID: ulid.Make().String(),
			Annotations:        groupAnnotations,
		}

		inputs, err := c.resolveCollectedInputs(ctx, posting, input)
		if err != nil {
			return nil, err
		}

		// Credit-only: if the wallet didn't cover the full accrual, issue advance and
		// move that slice through the advance-to-accrued path.
		if shortfall := input.Amount.Sub(collectedInputs(inputs).collectedFBOAmount()); c.shouldAdvanceShortfall(input, shortfall) {
			advanceInputs, err := c.advance.PlanIssue(ctx, advance.IssueInput{
				CustomerID:  c.customerID(input),
				ChargeID:    input.ChargeID,
				At:          input.BookedAt,
				Amount:      shortfall,
				Currency:    input.Currency,
				Filters:     input.Filters,
				TaxCode:     input.TaxCode,
				TaxBehavior: input.TaxBehavior,
			})
			if err != nil {
				return nil, err
			}

			inputs = append(inputs, advanceInputs...)
		}

		if len(inputs) == 0 {
			return nil, nil
		}

		for i, txInput := range inputs {
			if txInput != nil {
				inputs[i] = transactions.WithAnnotations(txInput, groupAnnotations)
			}
		}

		transactionGroup, err := c.ledger.CommitGroup(ctx, ledger.WithGroupID(transactions.GroupInputs(
			input.Namespace,
			groupAnnotations,
			inputs...,
		), posting.TransactionGroupID))
		if err != nil {
			return nil, fmt.Errorf("commit ledger transaction group: %w", err)
		}

		return collectedInputs(inputs).toCreditRealizations(input.ServicePeriod, transactionGroup.ID().ID), nil
	}

	return transaction.Run(ctx, c.transactionManager, run)
}

func (c *accrualCollector) collectToReceivable(ctx context.Context, input CollectToReceivableInput) (creditrealization.CreateAllocationInputs, error) {
	run := func(ctx context.Context) (creditrealization.CreateAllocationInputs, error) {
		if input.Amount.IsZero() {
			return nil, nil
		}

		groupAnnotations := input.Annotations
		if groupAnnotations == nil {
			groupAnnotations = ledger.ChargeAnnotations(models.NamespacedID{
				Namespace: input.Namespace,
				ID:        input.ChargeID,
			})
		}

		posting := breakage.PostingInput{
			TransactionGroupID: ulid.Make().String(),
			Annotations:        groupAnnotations,
		}

		inputs, err := c.resolveCoveredReceivableInputs(ctx, posting, input)
		if err != nil {
			return nil, err
		}
		if len(inputs) == 0 {
			return nil, nil
		}

		for i, txInput := range inputs {
			if txInput != nil {
				inputs[i] = transactions.WithAnnotations(txInput, groupAnnotations)
			}
		}

		transactionGroup, err := c.ledger.CommitGroup(ctx, ledger.WithGroupID(transactions.GroupInputs(
			input.Namespace,
			groupAnnotations,
			inputs...,
		), posting.TransactionGroupID))
		if err != nil {
			return nil, fmt.Errorf("commit ledger transaction group: %w", err)
		}

		return collectedInputs(inputs).toCreditRealizations(input.ServicePeriod, transactionGroup.ID().ID), nil
	}

	return transaction.Run(ctx, c.transactionManager, run)
}

func (c *accrualCollector) resolveCoveredReceivableInputs(ctx context.Context, posting breakage.PostingInput, input CollectToReceivableInput) ([]ledger.TransactionInput, error) {
	if err := ledger.ValidateTransactionAmount(input.Amount); err != nil {
		return nil, fmt.Errorf("amount: %w", err)
	}

	selections, err := c.collectCustomerFBOSelections(
		ctx,
		customer.CustomerID{Namespace: input.Namespace, ID: input.CustomerID},
		input.Currency,
		ledger.Route{Filters: input.Filters},
		input.Amount,
		input.SourceBalanceAsOf,
	)
	if err != nil {
		return nil, fmt.Errorf("collect customer FBO: %w", err)
	}

	if len(selections) == 0 {
		return nil, nil
	}

	for i := range selections {
		selections[i].collectionOriginID = lo.ToPtr(ulid.Make().String())
	}

	sources := fboCollectionSelections(selections).postingAmounts(&input.ChargeID)
	inputs, err := transactions.ResolveTransactions(
		ctx,
		c.deps,
		transactions.ResolutionScope{
			CustomerID: customer.CustomerID{Namespace: input.Namespace, ID: input.CustomerID},
			Namespace:  input.Namespace,
		},
		transactions.CoverCustomerReceivableTemplate{
			At:       input.BookedAt,
			Currency: input.Currency,
			Sources:  sources,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("resolve transactions: %w", err)
	}

	inputs, err = c.resolveCollectionBreakageInputs(ctx, collectionBreakageInput{
		Posting:    posting,
		ChargeID:   input.ChargeID,
		Selections: selections,
		Postings:   inputs,
	})
	if err != nil {
		return nil, err
	}

	return inputs, nil
}

func (c *accrualCollector) resolveCollectedInputs(ctx context.Context, posting breakage.PostingInput, input CollectToAccruedInput) ([]ledger.TransactionInput, error) {
	if err := ledger.ValidateTransactionAmount(input.Amount); err != nil {
		return nil, fmt.Errorf("amount: %w", err)
	}

	if err := input.Currency.Validate(); err != nil {
		return nil, fmt.Errorf("currency: %w", err)
	}

	if input.Currency.IsCustom() && !input.Currency.IsResolved() {
		return nil, fmt.Errorf("currency: custom currency must be resolved")
	}

	selections, err := c.collectCustomerFBOSelections(ctx, c.customerID(input), input.Currency, ledger.Route{Filters: input.Filters}, input.Amount, input.SourceBalanceAsOf)
	if err != nil {
		return nil, fmt.Errorf("collect customer FBO: %w", err)
	}

	if len(selections) == 0 {
		return nil, nil
	}

	for i := range selections {
		selections[i].collectionOriginID = lo.ToPtr(ulid.Make().String())
	}

	sources := fboCollectionSelections(selections).postingAmounts(&input.ChargeID)
	inputs, err := transactions.ResolveTransactions(
		ctx,
		c.deps,
		c.resolutionScope(input),
		transactions.TransferCustomerFBOToAccruedTemplate{
			At:          input.BookedAt,
			Currency:    input.Currency,
			TaxCode:     input.TaxCode,
			TaxBehavior: input.TaxBehavior,
			Sources:     sources,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("resolve transactions: %w", err)
	}

	inputs, err = c.resolveCollectionBreakageInputs(ctx, collectionBreakageInput{
		Posting:    posting,
		ChargeID:   input.ChargeID,
		Selections: selections,
		Postings:   inputs,
	})
	if err != nil {
		return nil, err
	}

	return inputs, nil
}

type sourcePostingIDs struct {
	transactionID string
	entryID       string
}

type collectionBreakageInput struct {
	Posting    breakage.PostingInput
	ChargeID   string
	Selections []fboCollectionSelection
	Postings   []ledger.TransactionInput
}

var _ models.Validator = collectionBreakageInput{}

func (i collectionBreakageInput) Validate() error {
	var errs []error

	if err := i.Posting.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.ChargeID == "" {
		errs = append(errs, errors.New("charge id is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c *accrualCollector) resolveCollectionBreakageInputs(ctx context.Context, input collectionBreakageInput) ([]ledger.TransactionInput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	inputs := input.Postings

	// Capture concrete source IDs while the resolved postings are still in hand.
	// A collection origin identifies exactly one selected FBO debit in this group.
	sourcesByOrigin := make(map[string]sourcePostingIDs, len(input.Selections))
	for idx, input := range inputs {
		identified, err := ledger.PreassignIDs(input)
		if err != nil {
			return nil, err
		}
		inputs[idx] = identified

		for _, entry := range identified.EntryInputs() {
			if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerFBO || !entry.Amount().IsNegative() {
				continue
			}
			originID := lo.FromPtr(entry.Provenance().CollectionOriginID)
			if _, exists := sourcesByOrigin[originID]; exists {
				return nil, fmt.Errorf("collection origin %s has multiple FBO source entries", originID)
			}
			sourcesByOrigin[originID] = sourcePostingIDs{transactionID: identified.AssignedID(), entryID: entry.AssignedID()}
		}
	}

	releaseRemainingByPlanID := make(map[string]alpacadecimal.Decimal)

	for _, selection := range input.Selections {
		if selection.source.breakagePlan == nil {
			continue
		}

		plan := *selection.source.breakagePlan
		remaining, ok := releaseRemainingByPlanID[plan.ID.ID]
		if !ok {
			remaining = plan.OpenAmount
		}

		// Legacy source-less plans can reserve multiple selected source slices,
		// so guard the aggregate release amount before writing release records.
		if selection.amount.GreaterThan(remaining) {
			return nil, fmt.Errorf("breakage release amount %s exceeds remaining plan amount %s for plan %s", selection.amount, remaining, plan.ID.ID)
		}

		releaseRemainingByPlanID[plan.ID.ID] = remaining.Sub(selection.amount)

		source, ok := sourcesByOrigin[lo.FromPtr(selection.collectionOriginID)]
		if !ok {
			return nil, fmt.Errorf("selected credit has no resolved FBO source posting")
		}

		releaseInput, err := c.breakage.ReleasePlan(ctx, breakage.ReleasePlanInput{
			PostingInput:        input.Posting,
			Plan:                plan,
			Amount:              selection.amount,
			SourceKind:          breakage.SourceKindUsage,
			SourceChargeID:      selection.source.sourceChargeID,
			SpendChargeID:       &input.ChargeID,
			CollectionOriginID:  selection.collectionOriginID,
			SourceTransactionID: &source.transactionID,
			SourceEntryID:       &source.entryID,
		})
		if err != nil {
			return nil, fmt.Errorf("resolve breakage release: %w", err)
		}

		inputs = append(inputs, releaseInput)
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

		annotations := maps.Clone(input.Annotations())
		if annotations == nil {
			annotations = make(models.Annotations)
		}

		annotations[ledger.AnnotationOriginTracked] = true
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
