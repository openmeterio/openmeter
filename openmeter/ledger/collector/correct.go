package collector

import (
	"context"
	"fmt"
	"sort"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type accrualCorrector struct {
	ledger ledger.Ledger
	deps   transactions.ResolverDependencies
}

// collectedSource is one logical “collection” in the group: the FBO→accrued
// forward tx, plus the receivable issue when that slice was advance-backed.
type collectedSource struct {
	transaction                       ledger.Transaction
	group                             ledger.TransactionGroup
	advanceReceivableIssueTransaction ledger.Transaction
}

type transactionCorrectionPlan struct {
	transaction ledger.Transaction
	group       ledger.TransactionGroup
	amount      alpacadecimal.Decimal
}

type plannedAction interface {
	isPlannedAction()
}

type plannedTransactionCorrection struct {
	transaction ledger.Transaction
	group       ledger.TransactionGroup
	amount      alpacadecimal.Decimal
}

func (plannedTransactionCorrection) isPlannedAction() {}

// plannedDirectInputs are inputs we already resolved (e.g. reissue); they skip
// the merge-and-CorrectTransaction path below.
type plannedDirectInputs struct {
	inputs []ledger.TransactionInput
}

func (plannedDirectInputs) isPlannedAction() {}

func (c *accrualCorrector) correct(ctx context.Context, input CorrectCollectedAccruedInput) (creditrealization.CreateCorrectionInputs, error) {
	if len(input.Corrections) == 0 {
		return nil, nil
	}

	// Plan first, execute later, so we can merge overlapping corrections cleanly.
	actions := make([]plannedAction, 0, len(input.Corrections))
	for _, correction := range input.Corrections {
		correctionActions, err := c.planCorrection(ctx, input, correction)
		if err != nil {
			return nil, err
		}
		actions = append(actions, correctionActions...)
	}

	resolvedInputs, err := c.resolvePlannedInputs(ctx, input, actions)
	if err != nil {
		return nil, err
	}
	if len(resolvedInputs) == 0 {
		return nil, nil
	}

	// Write the whole correction batch as one group and point every new correction
	// realization at that group.
	transactionGroup, err := c.ledger.CommitGroup(ctx, transactions.GroupInputs(
		input.Namespace,
		ledger.ChargeAnnotations(models.NamespacedID{
			Namespace: input.Namespace,
			ID:        input.ChargeID,
		}),
		resolvedInputs...,
	))
	if err != nil {
		return nil, fmt.Errorf("commit correction transaction group: %w", err)
	}

	out := make(creditrealization.CreateCorrectionInputs, 0, len(input.Corrections))
	for _, correction := range input.Corrections {
		out = append(out, creditrealization.CreateCorrectionInput{
			LedgerTransaction: ledgertransaction.GroupReference{
				TransactionGroupID: transactionGroup.ID().ID,
			},
			Amount:                correction.Amount,
			CorrectsRealizationID: correction.Allocation.ID,
		})
	}

	return out, nil
}

func (c *accrualCorrector) planCorrection(ctx context.Context, input CorrectCollectedAccruedInput, correction creditrealization.CorrectionRequestItem) ([]plannedAction, error) {
	originalGroup, err := c.originalGroup(ctx, input, correction)
	if err != nil {
		return nil, err
	}

	// SortHint maps the realization back to the original collected source in the group.
	source, err := c.collectedSourceBySortHint(originalGroup, correction.Allocation.SortHint)
	if err != nil {
		return nil, err
	}

	// Older data may not have lineage yet, so fall back to first-order source correction.
	if len(correction.Allocation.ActiveLineageSegments) == 0 {
		return plannedSourceCorrectionActions(source, correction.Amount.Abs(), source.advanceReceivableIssueTransaction != nil), nil
	}

	// Lineage tells us what this value looks like now, so consume that state first.
	remaining := correction.Amount.Abs()
	actions := make([]plannedAction, 0, len(correction.Allocation.ActiveLineageSegments)+2)
	segments := sortCorrectionSegments(correction.Allocation.ActiveLineageSegments)
	for _, segment := range segments {
		if !remaining.IsPositive() {
			break
		}

		segmentAmount := minDecimal(segment.Amount, remaining)
		if !segmentAmount.IsPositive() {
			continue
		}

		segmentActions, err := c.planSegmentCorrection(ctx, input, source, segment, segmentAmount)
		if err != nil {
			return nil, err
		}
		actions = append(actions, segmentActions...)

		remaining = remaining.Sub(segmentAmount)
	}

	if remaining.IsPositive() {
		return nil, fmt.Errorf("correction amount %s exceeds active lineage coverage for realization %s", correction.Amount.Abs().String(), correction.Allocation.ID)
	}

	return actions, nil
}

func (c *accrualCorrector) originalGroup(ctx context.Context, input CorrectCollectedAccruedInput, correction creditrealization.CorrectionRequestItem) (ledger.TransactionGroup, error) {
	group, err := c.ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: input.Namespace,
		ID:        correction.Allocation.LedgerTransaction.TransactionGroupID,
	})
	if err != nil {
		return nil, fmt.Errorf("get original transaction group %s: %w", correction.Allocation.LedgerTransaction.TransactionGroupID, err)
	}

	return group, nil
}

func (c *accrualCorrector) planSegmentCorrection(ctx context.Context, input CorrectCollectedAccruedInput, source collectedSource, segment creditrealization.ActiveLineageSegment, amount alpacadecimal.Decimal) ([]plannedAction, error) {
	// Each current segment state needs a slightly different unwind.
	switch segment.State {
	case creditrealization.LineageSegmentStateRealCredit:
		return plannedSourceCorrectionActions(source, amount, false), nil
	case creditrealization.LineageSegmentStateAdvanceUncovered:
		return plannedSourceCorrectionActions(source, amount, true), nil
	case creditrealization.LineageSegmentStateAdvanceBackfilled:
		return c.planBackfilledAdvanceSegment(ctx, input, source, segment, amount)
	default:
		return nil, fmt.Errorf("unsupported active lineage segment state %s", segment.State)
	}
}

func (c *accrualCorrector) planBackfilledAdvanceSegment(ctx context.Context, input CorrectCollectedAccruedInput, source collectedSource, segment creditrealization.ActiveLineageSegment, amount alpacadecimal.Decimal) ([]plannedAction, error) {
	if segment.BackingTransactionGroupID == nil || *segment.BackingTransactionGroupID == "" {
		return nil, fmt.Errorf("advance_backfilled segment missing backing transaction group id")
	}

	// Backfilled advance means we have to unwind both the later backfill and the
	// original advance-backed collection.
	backingGroup, err := c.ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: input.Namespace,
		ID:        *segment.BackingTransactionGroupID,
	})
	if err != nil {
		return nil, fmt.Errorf("get backing transaction group %s: %w", *segment.BackingTransactionGroupID, err)
	}

	actions := make([]plannedAction, 0, 4)
	if translateTx, err := c.forwardTransactionByTemplate(backingGroup, transactions.TemplateName(transactions.TranslateCustomerAccruedCostBasisTemplate{})); err == nil {
		actions = append(actions, plannedTransactionCorrection{
			transaction: translateTx,
			group:       backingGroup,
			amount:      amount,
		})
	}

	attributeTx, err := c.forwardTransactionByTemplate(backingGroup, transactions.TemplateName(transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{}))
	if err != nil {
		return nil, fmt.Errorf("find backing advance receivable attribution transaction in group %s: %w", backingGroup.ID().ID, err)
	}
	actions = append(actions, plannedTransactionCorrection{
		transaction: attributeTx,
		group:       backingGroup,
		amount:      amount,
	})
	actions = append(actions, plannedSourceCorrectionActions(source, amount, true)...)

	// The purchased-credit-covered part becomes available credit again.
	reissueInputs, err := c.reissueBackfilledCredit(ctx, input, backingGroup, amount)
	if err != nil {
		return nil, err
	}
	actions = append(actions, plannedDirectInputs{inputs: reissueInputs})

	return actions, nil
}

func (c *accrualCorrector) reissueBackfilledCredit(ctx context.Context, input CorrectCollectedAccruedInput, backingGroup ledger.TransactionGroup, amount alpacadecimal.Decimal) ([]ledger.TransactionInput, error) {
	// Re-issue into the same known-cost bucket the backfill had used.
	currency, costBasis, err := c.backfilledIssueRoute(backingGroup)
	if err != nil {
		return nil, err
	}

	resolved, err := transactions.ResolveTransactions(
		ctx,
		c.deps,
		transactions.ResolutionScope{
			CustomerID: customer.CustomerID{
				Namespace: input.Namespace,
				ID:        input.CustomerID,
			},
			Namespace: input.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:        input.AllocateAt,
			Amount:    amount,
			Currency:  currency,
			CostBasis: costBasis,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("resolve re-issued purchased credit: %w", err)
	}

	out := make([]ledger.TransactionInput, 0, len(resolved))
	for _, txInput := range resolved {
		out = append(out, transactions.WithAnnotations(txInput, ledger.TransactionAnnotations(
			transactions.TemplateName(transactions.IssueCustomerReceivableTemplate{}),
			ledger.TransactionDirectionCorrection,
		)))
	}

	return out, nil
}

func plannedSourceCorrectionActions(source collectedSource, amount alpacadecimal.Decimal, includeAdvanceReceivable bool) []plannedAction {
	// A source correction always offsets the original collection transaction itself.
	// Advance-backed collection also needs the companion receivable-issue correction
	// so the offset reduces the advance-side obligation instead of manufacturing credit.
	actions := []plannedAction{
		plannedTransactionCorrection{
			transaction: source.transaction,
			group:       source.group,
			amount:      amount,
		},
	}

	if includeAdvanceReceivable && source.advanceReceivableIssueTransaction != nil {
		actions = append(actions, plannedTransactionCorrection{
			transaction: source.advanceReceivableIssueTransaction,
			group:       source.group,
			amount:      amount,
		})
	}

	return actions
}

func (c *accrualCorrector) resolvePlannedInputs(ctx context.Context, input CorrectCollectedAccruedInput, actions []plannedAction) ([]ledger.TransactionInput, error) {
	// Merge by original transaction before executing, so template-specific correction
	// still sees one aggregated amount per source.
	mergedCorrections := make(map[string]*transactionCorrectionPlan, len(actions))
	correctionOrder := make([]string, 0, len(actions))
	out := make([]ledger.TransactionInput, 0, len(actions))

	for _, action := range actions {
		switch planned := action.(type) {
		case plannedTransactionCorrection:
			key := planned.transaction.ID().Namespace + ":" + planned.transaction.ID().ID
			if existing, ok := mergedCorrections[key]; ok {
				existing.amount = existing.amount.Add(planned.amount)
				continue
			}

			mergedCorrections[key] = &transactionCorrectionPlan{
				transaction: planned.transaction,
				group:       planned.group,
				amount:      planned.amount,
			}
			correctionOrder = append(correctionOrder, key)
		case plannedDirectInputs:
			out = append(out, planned.inputs...)
		default:
			return nil, fmt.Errorf("unsupported planned action %T", action)
		}
	}

	for _, key := range correctionOrder {
		transactionPlan := mergedCorrections[key]
		correctionInputs, err := transactions.CorrectTransaction(ctx, c.deps, transactions.CorrectionInput{
			At:                  input.AllocateAt,
			Amount:              transactionPlan.amount,
			OriginalTransaction: transactionPlan.transaction,
			OriginalGroup:       transactionPlan.group,
		})
		if err != nil {
			return nil, fmt.Errorf("correct transaction %s: %w", transactionPlan.transaction.ID().ID, err)
		}
		out = append(out, correctionInputs...)
	}

	return out, nil
}

func (c *accrualCorrector) collectedSourceBySortHint(group ledger.TransactionGroup, sortHint int) (collectedSource, error) {
	sources, err := c.collectedSourcesForGroup(group)
	if err != nil {
		return collectedSource{}, fmt.Errorf("map correction sources for group %s: %w", group.ID().ID, err)
	}

	if sortHint < 0 || sortHint >= len(sources) {
		return collectedSource{}, fmt.Errorf("allocation sort hint %d out of range for transaction group %s", sortHint, group.ID().ID)
	}

	return sources[sortHint], nil
}

func (c *accrualCorrector) collectedSourcesForGroup(group ledger.TransactionGroup) ([]collectedSource, error) {
	out := make([]collectedSource, 0)
	for _, transaction := range group.Transactions() {
		templateName, err := ledger.TransactionTemplateNameFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s template name: %w", transaction.ID().ID, err)
		}

		direction, err := ledger.TransactionDirectionFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s direction: %w", transaction.ID().ID, err)
		}
		if direction != ledger.TransactionDirectionForward {
			continue
		}

		// Advance-backed collection comes with a receivable issue in the same group.
		var advanceReceivableIssueTransaction ledger.Transaction
		if templateName == transactions.TemplateName(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}) {
			advanceReceivableIssueTransaction, err = c.forwardTransactionByTemplate(group, transactions.TemplateName(transactions.IssueCustomerReceivableTemplate{}))
			if err != nil {
				return nil, fmt.Errorf("find issue receivable companion in group %s: %w", group.ID().ID, err)
			}
		}

		for _, entry := range transaction.Entries() {
			if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO && entry.Amount().IsNegative() {
				out = append(out, collectedSource{
					transaction:                       transaction,
					group:                             group,
					advanceReceivableIssueTransaction: advanceReceivableIssueTransaction,
				})
			}
		}
	}

	return out, nil
}

func (c *accrualCorrector) forwardTransactionByTemplate(group ledger.TransactionGroup, templateName string) (ledger.Transaction, error) {
	for _, transaction := range group.Transactions() {
		currentTemplateName, err := ledger.TransactionTemplateNameFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s template name: %w", transaction.ID().ID, err)
		}

		direction, err := ledger.TransactionDirectionFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s direction: %w", transaction.ID().ID, err)
		}

		if currentTemplateName == templateName && direction == ledger.TransactionDirectionForward {
			return transaction, nil
		}
	}

	return nil, fmt.Errorf("transaction with template %s not found", templateName)
}

func (c *accrualCorrector) backfilledIssueRoute(group ledger.TransactionGroup) (currencyx.Code, *alpacadecimal.Decimal, error) {
	for _, transaction := range group.Transactions() {
		for _, entry := range transaction.Entries() {
			route := entry.PostingAddress().Route().Route()
			if route.CostBasis != nil {
				return route.Currency, route.CostBasis, nil
			}
		}
	}

	return "", nil, fmt.Errorf("backing transaction group %s does not contain a known cost basis route", group.ID().ID)
}

func sortCorrectionSegments(segments []creditrealization.ActiveLineageSegment) []creditrealization.ActiveLineageSegment {
	sorted := append([]creditrealization.ActiveLineageSegment(nil), segments...)
	sort.SliceStable(sorted, func(i, j int) bool {
		// Go from most downstream representation back outward.
		precedence := func(state creditrealization.LineageSegmentState) int {
			switch state {
			case creditrealization.LineageSegmentStateAdvanceBackfilled:
				return 0
			case creditrealization.LineageSegmentStateAdvanceUncovered:
				return 1
			case creditrealization.LineageSegmentStateRealCredit:
				return 2
			default:
				return 3
			}
		}

		return precedence(sorted[i].State) < precedence(sorted[j].State)
	})

	return sorted
}

func minDecimal(a, b alpacadecimal.Decimal) alpacadecimal.Decimal {
	if a.GreaterThan(b) {
		return b
	}

	return a
}
