package collector

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type accrualCorrector struct {
	ledger             ledger.Ledger
	advance            advance.Service
	deps               transactions.ResolverDependencies
	breakage           breakage.Service
	transactionManager transaction.Creator
}

// collectedSource is one logical “collection” in the group: the FBO→accrued
// forward tx, plus the receivable issue when that slice was advance-backed.
type collectedSource struct {
	entries                           []ledger.Entry
	transaction                       ledger.Transaction
	group                             ledger.TransactionGroup
	advanceReceivableIssueTransaction ledger.Transaction
}

type transactionCorrectionPlan struct {
	sourceEntryAmounts map[string]alpacadecimal.Decimal
	transaction        ledger.Transaction
	group              ledger.TransactionGroup
	amount             alpacadecimal.Decimal
}

type plannedAction interface {
	isPlannedAction()
}

type plannedTransactionCorrection struct {
	sourceEntryAmounts map[string]alpacadecimal.Decimal
	transaction        ledger.Transaction
	group              ledger.TransactionGroup
	amount             alpacadecimal.Decimal
}

func (plannedTransactionCorrection) isPlannedAction() {}

func (p plannedTransactionCorrection) mergeKey() string {
	return p.transaction.ID().Namespace + ":" + p.transaction.ID().ID
}

// plannedDirectInputs are inputs we already resolved (e.g. reissue); they skip
// the merge-and-CorrectTransaction path below.
type plannedDirectInputs struct {
	inputs          []ledger.TransactionInput
	breakagePending []breakage.PendingRecord
}

func (plannedDirectInputs) isPlannedAction() {}

type resolvedCorrectionInputs struct {
	inputs          []ledger.TransactionInput
	breakagePending []breakage.PendingRecord
}

func (c *accrualCorrector) correct(ctx context.Context, input CorrectCollectedAccruedInput) (creditrealization.CreateCorrectionInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	run := func(ctx context.Context) (creditrealization.CreateCorrectionInputs, error) {
		if len(input.Corrections) == 0 {
			return nil, nil
		}

		accounts, err := c.deps.AccountService.GetCustomerAccounts(ctx, customer.CustomerID{
			Namespace: input.Namespace,
			ID:        input.CustomerID,
		})
		if err != nil {
			return nil, err
		}

		if err := accounts.LockForPosting(ctx, c.deps.AccountCatalog); err != nil {
			return nil, err
		}

		// Legacy selection reserves entry amounts across the batch. Origin
		// corrections reconstruct only their indexed origin histories below.
		used := make(map[string]alpacadecimal.Decimal)

		for _, correction := range input.Corrections {
			if correction.Allocation.Annotations[ledger.AnnotationOriginTracked] != true {
				used, err = c.correctedSourceAmounts(ctx, input)
				if err != nil {
					return nil, err
				}

				break
			}
		}

		// Reserve source capacity across the batch before committing any postings.
		actions := make([]plannedAction, 0, len(input.Corrections))
		selectedSegments := make(map[string]map[string]alpacadecimal.Decimal)

		for _, correction := range input.Corrections {
			correctionActions, err := c.planCorrection(ctx, input, correction, used, selectedSegments)
			if err != nil {
				return nil, err
			}
			actions = append(actions, correctionActions...)
		}

		resolved, err := c.resolvePlannedInputs(ctx, input, actions)
		if err != nil {
			return nil, err
		}
		if len(resolved.inputs) == 0 {
			return nil, nil
		}

		groupAnnotations := input.Annotations
		if groupAnnotations == nil {
			groupAnnotations = ledger.ChargeAnnotations(models.NamespacedID{
				Namespace: input.Namespace,
				ID:        input.ChargeID,
			})
		}

		// Write the whole correction batch as one group and point every new correction
		// realization at that group.
		for i, txInput := range resolved.inputs {
			if txInput != nil {
				resolved.inputs[i] = transactions.WithAnnotations(txInput, groupAnnotations)
			}
		}

		transactionGroup, err := c.ledger.CommitGroup(ctx, transactions.GroupInputs(
			input.Namespace,
			groupAnnotations,
			resolved.inputs...,
		))
		if err != nil {
			return nil, fmt.Errorf("commit correction transaction group: %w", err)
		}

		if err := c.breakage.PersistCommittedRecords(ctx, resolved.breakagePending, transactionGroup); err != nil {
			return nil, fmt.Errorf("persist breakage records: %w", err)
		}

		out := make(creditrealization.CreateCorrectionInputs, 0, len(input.Corrections))
		for _, correction := range input.Corrections {
			var annotations models.Annotations

			if selected := selectedSegments[correction.Allocation.ID]; selected != nil {
				annotations, err = legacylineage.CorrectionAnnotations(selected)
				if err != nil {
					return nil, err
				}
			}

			if correction.Allocation.Annotations[ledger.AnnotationOriginTracked] == true {
				annotations = models.Annotations{ledger.AnnotationOriginTracked: true}
			}

			out = append(out, creditrealization.CreateCorrectionInput{
				Annotations: annotations,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: transactionGroup.ID().ID,
				},
				Amount:                correction.Amount,
				CorrectsRealizationID: correction.Allocation.ID,
			})
		}

		return out, nil
	}

	return transaction.Run(ctx, c.transactionManager, run)
}

func (c *accrualCorrector) planCorrection(ctx context.Context, input CorrectCollectedAccruedInput, correction creditrealization.CorrectionRequestItem, used map[string]alpacadecimal.Decimal, selectedSegments map[string]map[string]alpacadecimal.Decimal) ([]plannedAction, error) {
	originalGroup, err := c.originalGroup(ctx, input, correction)
	if err != nil {
		return nil, err
	}

	// SortHint maps the realization back to the original collected source in the group.
	source, err := c.collectedSourceBySortHint(originalGroup, correction.Allocation.SortHint)
	if err != nil {
		return nil, err
	}

	for _, entry := range source.transaction.Entries() {
		if entry.Provenance().CollectionOriginID != nil {
			return c.planOriginCorrection(ctx, input, source, correction.Amount.Abs())
		}
	}

	// Older data may not have lineage yet, so fall back to first-order source correction.
	segments := input.LineageSegmentsByRealization[correction.Allocation.ID]
	if len(segments) == 0 {
		return c.planUntrackedCorrection(ctx, input, source, correction.Amount.Abs(), used)
	}

	positions, evidence, err := c.readLegacyPositions(ctx, input, source, segments, used)
	if err != nil {
		return nil, err
	}

	selected, err := planCollectionCorrection(collectionCorrectionInput{
		amount:    correction.Amount.Abs(),
		positions: positions,
	})
	if err != nil {
		return nil, err
	}

	selectedSegments[correction.Allocation.ID] = make(map[string]alpacadecimal.Decimal)

	return c.writeLegacyCorrection(ctx, input, selected, evidence, used, selectedSegments[correction.Allocation.ID])
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

func (c *accrualCorrector) planSegmentCorrection(ctx context.Context, input CorrectCollectedAccruedInput, source collectedSource, segment legacylineage.Segment, amount alpacadecimal.Decimal, used map[string]alpacadecimal.Decimal) ([]plannedAction, error) {
	// Legacy segment state chooses posting mechanics only. The shared planner
	// has already selected the funding source and amount.
	switch segment.State {
	case creditrealization.LineageSegmentStateRealCredit,
		creditrealization.LineageSegmentStateReceivableCoverage:
		return plannedSourceCorrectionActions(source, amount, used)
	case creditrealization.LineageSegmentStateAdvanceUncovered:
		return c.planLegacyAdvanceCorrection(ctx, input, source, nil, amount)
	case creditrealization.LineageSegmentStateAdvanceBackfilled:
		if segment.BackingTransactionGroupID == nil {
			return nil, fmt.Errorf("advance_backfilled segment missing backing transaction group id")
		}

		return c.planLegacyAdvanceCorrection(ctx, input, source, segment.BackingTransactionGroupID, amount)
	case creditrealization.LineageSegmentStateEarningsRecognized:
		return c.planRecognizedEarningsSegment(ctx, input, source, segment, amount, used)
	default:
		return nil, fmt.Errorf("unsupported active lineage segment state %s", segment.State)
	}
}

func (c *accrualCorrector) planRecognizedEarningsSegment(ctx context.Context, input CorrectCollectedAccruedInput, source collectedSource, segment legacylineage.Segment, amount alpacadecimal.Decimal, used map[string]alpacadecimal.Decimal) ([]plannedAction, error) {
	if segment.BackingTransactionGroupID == nil || *segment.BackingTransactionGroupID == "" {
		return nil, fmt.Errorf("earnings_recognized segment missing backing transaction group id")
	}
	if segment.SourceState == nil {
		return nil, fmt.Errorf("earnings_recognized segment missing source state")
	}
	if *segment.SourceState == creditrealization.LineageSegmentStateEarningsRecognized {
		return nil, fmt.Errorf("earnings_recognized segment source state cannot be earnings_recognized")
	}

	recognitionGroup, err := c.ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: input.Namespace,
		ID:        *segment.BackingTransactionGroupID,
	})
	if err != nil {
		return nil, fmt.Errorf("get recognition transaction group %s: %w", *segment.BackingTransactionGroupID, err)
	}

	recognitionTx, err := c.forwardTransactionByTemplate(recognitionGroup, transactions.TemplateCode(transactions.RecognizeEarningsFromAttributableAccruedTemplate{}))
	if err != nil {
		return nil, fmt.Errorf("find recognition transaction in group %s: %w", recognitionGroup.ID().ID, err)
	}

	sourceSegment := segment
	sourceSegment.State = *segment.SourceState
	sourceSegment.BackingTransactionGroupID = segment.SourceBackingTransactionGroupID
	sourceSegment.SourceState = nil
	sourceSegment.SourceBackingTransactionGroupID = nil

	sourceActions, err := c.planSegmentCorrection(ctx, input, source, sourceSegment, amount, used)
	if err != nil {
		return nil, err
	}

	entryAmounts, err := c.recognizedSourceAmounts(ctx, recognizedSourceAmountsInput{
		correction: input, recognition: recognitionTx, actions: sourceActions, amount: amount, used: used,
	})
	if err != nil {
		return nil, err
	}
	actions := []plannedAction{
		plannedTransactionCorrection{
			sourceEntryAmounts: entryAmounts,
			transaction:        recognitionTx,
			group:              recognitionGroup,
			amount:             amount,
		},
	}
	actions = append(actions, sourceActions...)

	return actions, nil
}

func plannedSourceCorrectionActions(source collectedSource, amount alpacadecimal.Decimal, used map[string]alpacadecimal.Decimal) ([]plannedAction, error) {
	entryAmounts, err := reserveCorrectionSources(source.entries, amount, used)
	if err != nil {
		return nil, err
	}

	actions := []plannedAction{
		plannedTransactionCorrection{
			sourceEntryAmounts: entryAmounts,
			transaction:        source.transaction,
			group:              source.group,
			amount:             amount,
		},
	}

	return actions, nil
}

func (c *accrualCorrector) resolvePlannedInputs(ctx context.Context, input CorrectCollectedAccruedInput, actions []plannedAction) (resolvedCorrectionInputs, error) {
	// Merge by original transaction before executing, so template-specific correction
	// still sees one aggregated amount per source.
	mergedCorrections := make(map[string]*transactionCorrectionPlan, len(actions))
	correctionOrder := make([]string, 0, len(actions))
	out := make([]ledger.TransactionInput, 0, len(actions))
	breakagePending := make([]breakage.PendingRecord, 0)

	for _, action := range actions {
		switch planned := action.(type) {
		case plannedTransactionCorrection:
			key := planned.mergeKey()
			if existing, ok := mergedCorrections[key]; ok {
				if (existing.sourceEntryAmounts == nil) != (planned.sourceEntryAmounts == nil) {
					return resolvedCorrectionInputs{}, fmt.Errorf("cannot merge scoped and unscoped corrections of transaction %s", key)
				}
				existing.amount = existing.amount.Add(planned.amount)
				for id, amount := range planned.sourceEntryAmounts {
					existing.sourceEntryAmounts[id] = existing.sourceEntryAmounts[id].Add(amount)
				}
				continue
			}

			mergedCorrections[key] = &transactionCorrectionPlan{
				sourceEntryAmounts: maps.Clone(planned.sourceEntryAmounts),
				transaction:        planned.transaction,
				group:              planned.group,
				amount:             planned.amount,
			}
			correctionOrder = append(correctionOrder, key)
		case plannedDirectInputs:
			out = append(out, planned.inputs...)
			breakagePending = append(breakagePending, planned.breakagePending...)
		default:
			return resolvedCorrectionInputs{}, fmt.Errorf("unsupported planned action %T", action)
		}
	}

	for _, key := range correctionOrder {
		transactionPlan := mergedCorrections[key]
		breakageInputs, pending, err := c.resolveBreakageReopenInputs(ctx, input, *transactionPlan)
		if err != nil {
			return resolvedCorrectionInputs{}, err
		}
		out = append(out, breakageInputs...)
		breakagePending = append(breakagePending, pending...)

		correctionInputs, err := transactions.CorrectTransaction(ctx, c.deps, transactions.CorrectionInput{
			At:                  input.AllocateAt,
			Amount:              transactionPlan.amount,
			SourceEntryAmounts:  transactionPlan.sourceEntryAmounts,
			OriginalTransaction: transactionPlan.transaction,
			OriginalGroup:       transactionPlan.group,
		})
		if err != nil {
			return resolvedCorrectionInputs{}, fmt.Errorf("correct transaction %s: %w", transactionPlan.transaction.ID().ID, err)
		}
		out = append(out, correctionInputs...)
	}

	return resolvedCorrectionInputs{
		inputs:          out,
		breakagePending: breakagePending,
	}, nil
}

type correctedFBOEntry struct {
	entry  ledger.Entry
	amount alpacadecimal.Decimal
}

func (c *accrualCorrector) resolveBreakageReopenInputs(ctx context.Context, input CorrectCollectedAccruedInput, transactionPlan transactionCorrectionPlan) ([]ledger.TransactionInput, []breakage.PendingRecord, error) {
	templateCode, err := ledger.TransactionTemplateCodeFromAnnotations(transactionPlan.transaction.Annotations())
	if err != nil {
		return nil, nil, fmt.Errorf("transaction %s template code: %w", transactionPlan.transaction.ID().ID, err)
	}
	if templateCode != transactions.TemplateCode(transactions.TransferCustomerFBOToAccruedTemplate{}) &&
		templateCode != transactions.TemplateCode(transactions.CoverCustomerReceivableTemplate{}) {
		return nil, nil, nil
	}

	correctedEntries := correctedFBOEntriesForAmount(transactionPlan.transaction, transactionPlan.amount)
	if transactionPlan.sourceEntryAmounts != nil {
		correctedEntries = nil
		for _, entry := range transactionPlan.transaction.Entries() {
			if amount := transactionPlan.sourceEntryAmounts[entry.ID().ID]; amount.IsPositive() {
				correctedEntries = append(correctedEntries, correctedFBOEntry{entry: entry, amount: amount})
			}
		}
	}
	if len(correctedEntries) == 0 {
		return nil, nil, nil
	}

	sourceEntryIDs := make([]string, 0, len(correctedEntries))
	for _, correctedEntry := range correctedEntries {
		sourceEntryIDs = append(sourceEntryIDs, correctedEntry.entry.ID().ID)
	}

	releases, err := c.breakage.ListReleases(ctx, breakage.ListReleasesInput{
		CustomerID:    customer.CustomerID{Namespace: input.Namespace, ID: input.CustomerID},
		SourceEntryID: sourceEntryIDs,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("list breakage releases: %w", err)
	}

	releasesBySourceEntryID := make(map[string][]breakage.Release, len(releases))
	for _, release := range releases {
		if release.SourceEntryID == nil || *release.SourceEntryID == "" {
			continue
		}

		releasesBySourceEntryID[*release.SourceEntryID] = append(releasesBySourceEntryID[*release.SourceEntryID], release)
	}

	inputs := make([]ledger.TransactionInput, 0, len(releases))
	pending := make([]breakage.PendingRecord, 0, len(releases))
	for _, correctedEntry := range correctedEntries {
		remaining := correctedEntry.amount
		for _, release := range releasesBySourceEntryID[correctedEntry.entry.ID().ID] {
			if !remaining.IsPositive() {
				break
			}

			amount := minDecimal(release.OpenAmount, remaining)
			if !amount.IsPositive() {
				continue
			}

			reopenInput, reopenRecord, err := c.breakage.ReopenRelease(ctx, breakage.ReopenReleaseInput{
				Release:            release,
				Amount:             amount,
				SourceKind:         breakage.SourceKindUsageCorrection,
				SourceChargeID:     correctedEntry.entry.Provenance().SourceChargeID,
				SpendChargeID:      correctedEntry.entry.Provenance().SpendChargeID,
				CollectionOriginID: correctedEntry.entry.Provenance().CollectionOriginID,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("resolve breakage reopen: %w", err)
			}

			inputs = append(inputs, reopenInput)
			pending = append(pending, reopenRecord)
			remaining = remaining.Sub(amount)
		}
	}

	return inputs, pending, nil
}

func correctedFBOEntriesForAmount(transaction ledger.Transaction, amount alpacadecimal.Decimal) []correctedFBOEntry {
	sourceEntries := make([]ledger.Entry, 0)
	for _, entry := range transaction.Entries() {
		if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO && entry.Amount().IsNegative() {
			sourceEntries = append(sourceEntries, entry)
		}
	}

	sort.SliceStable(sourceEntries, func(i, j int) bool {
		return compareCollectedFBOCorrectionSourceEntries(sourceEntries[i], sourceEntries[j]) < 0
	})

	remaining := amount
	out := make([]correctedFBOEntry, 0, len(sourceEntries))
	for idx := len(sourceEntries) - 1; idx >= 0 && remaining.IsPositive(); idx-- {
		entry := sourceEntries[idx]
		entryAmount := entry.Amount().Abs()
		if entryAmount.GreaterThan(remaining) {
			entryAmount = remaining
		}

		out = append(out, correctedFBOEntry{
			entry:  entry,
			amount: entryAmount,
		})
		remaining = remaining.Sub(entryAmount)
	}

	return out
}

func compareCollectedFBOCorrectionSourceEntries(left ledger.Entry, right ledger.Entry) int {
	leftOrder, leftHasOrder := left.Annotations().GetInt(ledger.AnnotationCollectionSourceOrder)
	rightOrder, rightHasOrder := right.Annotations().GetInt(ledger.AnnotationCollectionSourceOrder)
	if leftHasOrder && rightHasOrder && leftOrder != rightOrder {
		return cmp.Compare(leftOrder, rightOrder)
	}

	leftPriority := customerFBOPriority(left.PostingAddress().Route().Route())
	rightPriority := customerFBOPriority(right.PostingAddress().Route().Route())
	if leftPriority != rightPriority {
		return cmp.Compare(leftPriority, rightPriority)
	}

	if c := cmp.Compare(left.PostingAddress().SubAccountID(), right.PostingAddress().SubAccountID()); c != 0 {
		return c
	}

	return cmp.Compare(left.IdentityKey(), right.IdentityKey())
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
		templateCode, err := ledger.TransactionTemplateCodeFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s template code: %w", transaction.ID().ID, err)
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
		if templateCode == transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}) {
			advanceReceivableIssueTransaction, err = c.forwardTransactionByTemplate(group, transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}))
			if err != nil {
				return nil, fmt.Errorf("find issue receivable companion in group %s: %w", group.ID().ID, err)
			}
		}

		sourceIndexBySubAccount := make(map[string]int)
		for _, entry := range transaction.Entries() {
			if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO && entry.Amount().IsNegative() {
				subAccountID := entry.PostingAddress().SubAccountID()
				if idx, ok := sourceIndexBySubAccount[subAccountID]; ok {
					out[idx].entries = append(out[idx].entries, entry)
					continue
				}
				sourceIndexBySubAccount[subAccountID] = len(out)
				out = append(out, collectedSource{
					entries:                           []ledger.Entry{entry},
					transaction:                       transaction,
					group:                             group,
					advanceReceivableIssueTransaction: advanceReceivableIssueTransaction,
				})
			}
		}
	}

	return out, nil
}

func (c *accrualCorrector) forwardTransactionByTemplate(group ledger.TransactionGroup, templateCode string) (ledger.Transaction, error) {
	for _, transaction := range group.Transactions() {
		currentTemplateCode, err := ledger.TransactionTemplateCodeFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s template code: %w", transaction.ID().ID, err)
		}

		direction, err := ledger.TransactionDirectionFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s direction: %w", transaction.ID().ID, err)
		}

		if currentTemplateCode == templateCode && direction == ledger.TransactionDirectionForward {
			return transaction, nil
		}
	}

	return nil, fmt.Errorf("transaction with template code %s not found", templateCode)
}

func minDecimal(a, b alpacadecimal.Decimal) alpacadecimal.Decimal {
	if a.GreaterThan(b) {
		return b
	}

	return a
}

// correctedSourceAmounts reads immutable correction links so a later partial
// correction resumes the original source suffix instead of restoring it twice.
func (c *accrualCorrector) correctedSourceAmounts(ctx context.Context, input CorrectCollectedAccruedInput) (map[string]alpacadecimal.Decimal, error) {
	out := make(map[string]alpacadecimal.Decimal)
	var cursor *ledger.TransactionCursor
	for {
		page, err := c.ledger.ListTransactions(ctx, ledger.ListTransactionsInput{
			Namespace: input.Namespace, Limit: 100, Cursor: cursor,
			AnnotationFilters: map[string]string{
				ledger.AnnotationChargeID:             input.ChargeID,
				ledger.AnnotationTransactionDirection: string(ledger.TransactionDirectionCorrection),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("list prior charge corrections: %w", err)
		}
		for _, tx := range page.Items {
			for _, entry := range tx.Entries() {
				if !entry.Amount().IsPositive() {
					continue
				}
				_, identity, err := ledger.EntryIdentityKeyText(entry.IdentityKey()).Parse()
				if err != nil {
					return nil, fmt.Errorf("parse correction entry identity: %w", err)
				}
				if identity.CorrectionSource != nil {
					out[*identity.CorrectionSource] = out[*identity.CorrectionSource].Add(entry.Amount())
				}
			}
		}
		if page.NextCursor == nil {
			return out, nil
		}
		cursor = page.NextCursor
	}
}

// reserveCorrectionSources resumes the reverse original collection order after
// subtracting both persisted corrections and reservations earlier in this batch.
func reserveCorrectionSources(entries []ledger.Entry, amount alpacadecimal.Decimal, used map[string]alpacadecimal.Decimal) (map[string]alpacadecimal.Decimal, error) {
	entries = slices.Clone(entries)
	slices.SortStableFunc(entries, compareCollectedFBOCorrectionSourceEntries)
	out := make(map[string]alpacadecimal.Decimal)
	remaining := amount
	for i := len(entries) - 1; i >= 0 && remaining.IsPositive(); i-- {
		entry := entries[i]
		available := entry.Amount().Abs().Sub(used[entry.ID().ID])
		if !available.IsPositive() {
			continue
		}
		selected := minDecimal(available, remaining)
		out[entry.ID().ID] = selected
		used[entry.ID().ID] = used[entry.ID().ID].Add(selected)
		remaining = remaining.Sub(selected)
	}
	if remaining.IsPositive() {
		return nil, fmt.Errorf("correction exceeds remaining source capacity by %s", remaining)
	}
	return out, nil
}

// Recognition groups can include multiple spends and cost bases. Reverse only
// the accrued buckets that the source unwind will debit, including a backfill's
// translation from unknown to known cost basis.
type recognizedSourceAmountsInput struct {
	correction  CorrectCollectedAccruedInput
	recognition ledger.Transaction
	actions     []plannedAction
	amount      alpacadecimal.Decimal
	used        map[string]alpacadecimal.Decimal
}

func (c *accrualCorrector) recognizedSourceAmounts(ctx context.Context, input recognizedSourceAmountsInput) (map[string]alpacadecimal.Decimal, error) {
	accrued := make(map[correctionPostingKey]alpacadecimal.Decimal)
	for _, action := range input.actions {
		var inputs []ledger.TransactionInput
		switch planned := action.(type) {
		case plannedTransactionCorrection:
			var err error
			inputs, err = transactions.CorrectTransaction(ctx, c.deps, transactions.CorrectionInput{
				At: input.correction.AllocateAt, Amount: planned.amount, OriginalTransaction: planned.transaction, OriginalGroup: planned.group, SourceEntryAmounts: planned.sourceEntryAmounts,
			})
			if err != nil {
				return nil, fmt.Errorf("resolve recognition source unwind: %w", err)
			}
		case plannedDirectInputs:
			inputs = planned.inputs
		default:
			return nil, fmt.Errorf("unsupported recognition source action %T", action)
		}
		for _, tx := range inputs {
			for _, entry := range tx.EntryInputs() {
				if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerAccrued {
					key := correctionEntryKey(entry)
					accrued[key] = accrued[key].Sub(entry.Amount())
				}
			}
		}
	}
	out := make(map[string]alpacadecimal.Decimal)
	selected := alpacadecimal.Zero
	for _, entry := range input.recognition.Entries() {
		if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerAccrued || !entry.Amount().IsNegative() {
			continue
		}
		key := correctionEntryKey(entry)
		required := accrued[key]
		available := entry.Amount().Abs().Sub(input.used[entry.ID().ID])
		if !required.IsPositive() || !available.IsPositive() {
			continue
		}
		take := minDecimal(required, available)
		out[entry.ID().ID] = take
		input.used[entry.ID().ID] = input.used[entry.ID().ID].Add(take)
		accrued[key] = required.Sub(take)
		selected = selected.Add(take)
	}
	if !selected.Equal(input.amount) {
		return nil, fmt.Errorf("recognition source coverage %s does not match correction amount %s", selected, input.amount)
	}
	for _, remaining := range accrued {
		if !remaining.IsZero() {
			return nil, fmt.Errorf("recognition correction leaves unmatched accrued amount %s", remaining)
		}
	}
	return out, nil
}

type correctionPostingKey struct{ subAccountID, sourceChargeID, spendChargeID string }

func correctionEntryKey(entry ledger.EntryInput) correctionPostingKey {
	return correctionPostingKey{
		subAccountID:   entry.PostingAddress().SubAccountID(),
		sourceChargeID: lo.FromPtrOr(entry.Provenance().SourceChargeID, ""),
		spendChargeID:  lo.FromPtrOr(entry.Provenance().SpendChargeID, ""),
	}
}
