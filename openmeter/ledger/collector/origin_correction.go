package collector

import (
	"context"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

func (c *accrualCorrector) planOriginCorrection(ctx context.Context, input CorrectCollectedAccruedInput, source collectedSource, amount alpacadecimal.Decimal) ([]plannedAction, error) {
	entries := make([]ledger.Entry, 0)
	for _, entry := range source.transaction.Entries() {
		if entry.PostingAddress().SubAccountID() == source.fboSubAccountID && entry.Amount().IsNegative() {
			if lo.FromPtr(entry.SpendChargeID()) != input.ChargeID {
				return nil, fmt.Errorf("collection origin belongs to a different spend charge")
			}
			if entry.OriginID() == nil {
				return nil, fmt.Errorf("collection mixes tracked and legacy source entries")
			}
			entries = append(entries, entry)
		}
	}
	slices.SortStableFunc(entries, compareCollectedFBOCorrectionSourceEntries)
	remaining := amount
	var actions []plannedAction
	for idx := len(entries) - 1; idx >= 0 && remaining.IsPositive(); idx-- {
		history, err := c.loadOriginHistory(ctx, input.Namespace, *entries[idx].OriginID())
		if err != nil {
			return nil, err
		}
		original, err := history.pairForTransaction(source.transaction.ID().ID)
		if err != nil {
			return nil, err
		}
		take := minDecimal(remaining, original.remaining)
		if !take.IsPositive() {
			continue
		}
		resolved, err := c.unwindOrigin(ctx, input, source, history, original, take)
		if err != nil {
			return nil, err
		}
		actions = append(actions, plannedDirectInputs(resolved))
		remaining = remaining.Sub(take)
	}
	if remaining.IsPositive() {
		return nil, fmt.Errorf("correction exceeds remaining origin balance by %s", remaining)
	}
	return actions, nil
}

func (c *accrualCorrector) unwindOrigin(ctx context.Context, input CorrectCollectedAccruedInput, source collectedSource, history originHistory, original *originPair, amount alpacadecimal.Decimal) (resolvedCorrectionInputs, error) {
	var out resolvedCorrectionInputs
	backingBySource := make(map[string]*originPair)
	backingOrder := make([]string, 0)
	recognizedBySource := make(map[string]alpacadecimal.Decimal)
	for _, pair := range history.pairs {
		switch pair.code {
		case transactions.TemplateCode(transactions.TranslateCustomerAccruedCostBasisTemplate{}):
			key := lo.FromPtr(pair.credit.SourceChargeID())
			if backingBySource[key] != nil {
				return out, fmt.Errorf("ambiguous backfill source for collection origin")
			}
			backingBySource[key] = pair
			backingOrder = append(backingOrder, key)
		case transactions.TemplateCode(transactions.RecognizeEarningsFromAttributableAccruedTemplate{}):
			key := lo.FromPtr(pair.credit.SourceChargeID())
			recognizedBySource[key] = recognizedBySource[key].Add(pair.remaining)
		}
	}
	backingAmounts := make(map[string]alpacadecimal.Decimal)
	remaining := amount
	for _, pair := range history.pairs {
		if pair.code != transactions.TemplateCode(transactions.RecognizeEarningsFromAttributableAccruedTemplate{}) || !remaining.IsPositive() {
			continue
		}
		take := minDecimal(pair.remaining, remaining)
		if !take.IsPositive() {
			continue
		}
		key := lo.FromPtr(pair.credit.SourceChargeID())
		if source.advanceReceivableIssueTransaction != nil {
			if backingBySource[key] == nil {
				return out, fmt.Errorf("recognized advance has no matching backfill")
			}
			backingAmounts[key] = backingAmounts[key].Add(take)
		} else if key != lo.FromPtr(original.debit.SourceChargeID()) {
			return out, fmt.Errorf("recognition source does not match original collection")
		}
		reversal, err := reverseOriginPair(input, pair, take)
		if err != nil {
			return out, err
		}
		out.inputs = append(out.inputs, reversal)
		remaining = remaining.Sub(take)
	}
	if source.advanceReceivableIssueTransaction != nil {
		for _, key := range backingOrder {
			pair := backingBySource[key]
			unrecognized := pair.remaining.Sub(recognizedBySource[key])
			if unrecognized.IsNegative() {
				return out, fmt.Errorf("recognition exceeds its matching backfill")
			}
			take := minDecimal(unrecognized, remaining)
			backingAmounts[key] = backingAmounts[key].Add(take)
			remaining = remaining.Sub(take)
		}
		backfilled := alpacadecimal.Zero
		for _, pair := range backingBySource {
			backfilled = backfilled.Add(pair.remaining)
		}
		if remaining.GreaterThan(original.remaining.Sub(backfilled)) {
			return out, fmt.Errorf("correction exceeds uncovered advance balance")
		}
		for _, key := range backingOrder {
			take := backingAmounts[key]
			if !take.IsPositive() {
				continue
			}
			resolved, err := c.unwindOriginBackfill(ctx, input, history, backingBySource[key], take)
			if err != nil {
				return out, err
			}
			out.inputs = append(out.inputs, resolved.inputs...)
			out.breakagePending = append(out.breakagePending, resolved.breakagePending...)
		}
	}
	reversal, err := reverseOriginPair(input, original, amount)
	if err != nil {
		return out, err
	}
	out.inputs = append(out.inputs, reversal)
	if source.advanceReceivableIssueTransaction != nil {
		issue, err := history.pairForTransaction(source.advanceReceivableIssueTransaction.ID().ID)
		if err != nil {
			return out, err
		}
		reversal, err := reverseOriginPair(input, issue, amount)
		if err != nil {
			return out, err
		}
		out.inputs = append(out.inputs, reversal)
	} else {
		// Restrict breakage reopening to this exact original source entry.
		plan := transactionCorrectionPlan{transaction: originTransactionView{Transaction: original.transaction, entries: []ledger.Entry{original.debit, original.credit}}, group: source.group, amount: amount}
		inputs, pending, err := c.resolveBreakageReopenInputs(ctx, input, plan)
		if err != nil {
			return out, err
		}
		out.inputs = append(out.inputs, inputs...)
		out.breakagePending = append(out.breakagePending, pending...)
	}
	return out, nil
}

func reverseOriginPair(input CorrectCollectedAccruedInput, pair *originPair, amount alpacadecimal.Decimal) (ledger.TransactionInput, error) {
	if amount.GreaterThan(pair.remaining) {
		return nil, fmt.Errorf("reversal exceeds remaining original entry amount")
	}
	return transactions.ReverseOriginEntryPair(transactions.ReverseOriginEntryPairInput{
		At: input.AllocateAt, Amount: amount, Transaction: pair.transaction, Debit: pair.debit, Credit: pair.credit,
	})
}

func (c *accrualCorrector) unwindOriginBackfill(ctx context.Context, input CorrectCollectedAccruedInput, history originHistory, backfill *originPair, amount alpacadecimal.Decimal) (resolvedCorrectionInputs, error) {
	var out resolvedCorrectionInputs
	var attribution *originPair
	for _, pair := range history.pairs {
		if pair.code == transactions.TemplateCode(transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{}) &&
			pair.transaction.GroupID() == backfill.transaction.GroupID() &&
			lo.FromPtr(pair.debit.SourceChargeID()) == lo.FromPtr(backfill.credit.SourceChargeID()) {
			if attribution != nil {
				return out, fmt.Errorf("ambiguous advance attribution for origin")
			}
			attribution = pair
		}
	}
	if attribution == nil {
		return out, fmt.Errorf("advance backfill has no matching receivable attribution")
	}
	for _, pair := range []*originPair{backfill, attribution} {
		reversal, err := reverseOriginPair(input, pair, amount)
		if err != nil {
			return out, err
		}
		out.inputs = append(out.inputs, reversal)
	}
	group, err := c.ledger.GetTransactionGroup(ctx, backfill.transaction.GroupID())
	if err != nil {
		return out, err
	}
	if c.breakage != nil {
		releases, err := c.breakage.ListReleases(ctx, breakage.ListReleasesInput{
			CustomerID:               customer.CustomerID{Namespace: input.Namespace, ID: input.CustomerID},
			SourceTransactionGroupID: []string{group.ID().ID}, ReleaseSourceKind: []breakage.SourceKind{breakage.SourceKindAdvanceBackfill},
		})
		if err != nil {
			return out, err
		}
		matching := make(map[string]bool)
		for _, tx := range history.transactions {
			for _, entry := range tx.Entries() {
				if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO &&
					lo.FromPtr(entry.OriginID()) == lo.FromPtr(backfill.credit.OriginID()) &&
					lo.FromPtr(entry.SourceChargeID()) == lo.FromPtr(backfill.credit.SourceChargeID()) {
					matching[tx.ID().ID] = true
				}
			}
		}
		remaining := amount
		for _, release := range releases {
			if !matching[release.BreakageTransactionID] || !remaining.IsPositive() {
				continue
			}
			take := minDecimal(remaining, release.OpenAmount)
			if !take.IsPositive() {
				continue
			}
			reopened, pending, err := c.breakage.ReopenRelease(ctx, breakage.ReopenReleaseInput{
				Release: release, Amount: take, SourceKind: breakage.SourceKindUsageCorrection,
				SourceChargeID: backfill.credit.SourceChargeID(), SpendChargeID: backfill.credit.SpendChargeID(), OriginID: backfill.credit.OriginID(),
			})
			if err != nil {
				return out, err
			}
			out.inputs = append(out.inputs, reopened)
			out.breakagePending = append(out.breakagePending, pending)
			remaining = remaining.Sub(take)
		}
	}
	// The attributed receivable retains the purchase's feature restrictions.
	// Priority is preserved explicitly even when a purchase was fully backfilled
	// and therefore never wrote an ordinary FBO issuance entry.
	route := attribution.debit.PostingAddress().Route().Route()
	priority, ok := group.Annotations().GetInt(ledger.AnnotationBackfillCreditPriority)
	if !ok {
		return out, fmt.Errorf("origin backfill is missing purchased credit priority")
	}
	reissued, err := transactions.ResolveTransactions(ctx, c.deps, transactions.ResolutionScope{
		CustomerID: customer.CustomerID{Namespace: input.Namespace, ID: input.CustomerID}, Namespace: input.Namespace,
	}, transactions.IssueCustomerReceivableTemplate{
		At: input.AllocateAt, Amount: amount, Currency: route.Currency, CostBasisCurrency: route.CostBasisCurrency,
		CostBasis: route.CostBasis, Features: route.Features, CreditPriority: &priority, SourceChargeID: attribution.debit.SourceChargeID(),
	})
	if err != nil {
		return out, err
	}
	for _, tx := range reissued {
		out.inputs = append(out.inputs, transactions.WithAnnotations(tx, ledger.TransactionAnnotations(
			transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}), ledger.TransactionDirectionCorrection)))
	}
	return out, nil
}

type originTransactionView struct {
	ledger.Transaction
	entries []ledger.Entry
}

func (v originTransactionView) Entries() []ledger.Entry { return v.entries }
