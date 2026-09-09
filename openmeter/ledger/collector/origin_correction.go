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
	for _, entry := range source.entries {
		if lo.FromPtr(entry.SpendChargeID()) != input.ChargeID {
			return nil, fmt.Errorf("collection origin belongs to a different spend charge")
		}
		if entry.CollectionOriginID() == nil {
			return nil, fmt.Errorf("collection mixes tracked and legacy source entries")
		}
		entries = append(entries, entry)
	}

	slices.SortStableFunc(entries, compareCollectedFBOCorrectionSourceEntries)
	histories := make(map[string]originReferences)
	positionsByOrigin := make(map[string][]correctionPosition)
	originals := make(map[string]*originPair)
	var origins []correctionPosition
	for idx, entry := range entries {
		id := *entry.CollectionOriginID()
		history, err := c.loadOriginReferences(ctx, input.Namespace, id)
		if err != nil {
			return nil, err
		}
		original, err := history.pairForTransaction(source.transaction.ID().ID)
		if err != nil {
			return nil, err
		}
		positions, err := c.readOriginPositions(ctx, input, id, history, original)
		if err != nil {
			return nil, err
		}
		position := correctionPosition{id: id, order: idx}
		for _, p := range positions {
			position.accrued = position.accrued.Add(p.amount())
		}
		origins = append(origins, position)
		histories[id], positionsByOrigin[id], originals[id] = history, positions, original
	}
	selected, err := planCollectionCorrection(collectionCorrectionInput{amount: amount, positions: origins})
	if err != nil {
		return nil, fmt.Errorf("exceeds remaining origin balance: %w", err)
	}
	var actions []plannedAction
	for _, selection := range selected {
		resolved, err := c.unwindOrigin(ctx, input, source, histories[selection.id], originals[selection.id], positionsByOrigin[selection.id], selection.amount)
		if err != nil {
			return nil, err
		}
		actions = append(actions, plannedDirectInputs(resolved))
	}
	return actions, nil
}

func (c *accrualCorrector) unwindOrigin(ctx context.Context, input CorrectCollectedAccruedInput, source collectedSource, history originReferences, original *originPair, positions []correctionPosition, amount alpacadecimal.Decimal) (resolvedCorrectionInputs, error) {
	var out resolvedCorrectionInputs
	selections, err := planCollectionCorrection(collectionCorrectionInput{amount: amount, positions: positions})
	if err != nil {
		return out, err
	}
	for _, selection := range selections {
		remainingRecognition := selection.earnings
		for _, pair := range history.pairs {
			if pair.role != originRoleRecognition || lo.FromPtr(pair.credit.SourceChargeID()) != selection.id {
				continue
			}
			take := minDecimal(remainingRecognition, pair.remaining)
			if !take.IsPositive() {
				continue
			}
			reversal, err := reverseOriginPair(input, pair, take)
			if err != nil {
				return out, err
			}
			out.inputs = append(out.inputs, reversal)
			remainingRecognition = remainingRecognition.Sub(take)
		}
		if remainingRecognition.IsPositive() {
			return out, fmt.Errorf("earnings position lacks original recognition references")
		}
		if source.advanceReceivableIssueTransaction == nil || selection.id == unknownOriginSource {
			continue
		}
		remainingBacking := selection.amount
		for _, pair := range history.pairs {
			if pair.role != originRoleBacking || lo.FromPtr(pair.credit.SourceChargeID()) != selection.id {
				continue
			}
			take := minDecimal(remainingBacking, pair.remaining)
			if !take.IsPositive() {
				continue
			}
			resolved, err := c.unwindOriginBackfill(ctx, input, history, pair, take)
			if err != nil {
				return out, err
			}
			out.inputs = append(out.inputs, resolved.inputs...)
			out.breakagePending = append(out.breakagePending, resolved.breakagePending...)
			remainingBacking = remainingBacking.Sub(take)
		}
		if remainingBacking.IsPositive() {
			return out, fmt.Errorf("funded position lacks original backing references")
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

func (c *accrualCorrector) unwindOriginBackfill(ctx context.Context, input CorrectCollectedAccruedInput, history originReferences, backfill *originPair, amount alpacadecimal.Decimal) (resolvedCorrectionInputs, error) {
	var out resolvedCorrectionInputs
	var attribution *originPair
	for _, pair := range history.pairs {
		if pair.role == originRoleAttribution &&
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
					lo.FromPtr(entry.CollectionOriginID()) == lo.FromPtr(backfill.credit.CollectionOriginID()) &&
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
				SourceChargeID: backfill.credit.SourceChargeID(), SpendChargeID: backfill.credit.SpendChargeID(), CollectionOriginID: backfill.credit.CollectionOriginID(),
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
