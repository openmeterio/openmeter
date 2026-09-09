package collector

import (
	"context"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/models"
)

type legacyCorrectionPart struct {
	segment legacylineage.Segment
	amount  alpacadecimal.Decimal
}

type legacyCorrectionEvidence struct {
	source collectedSource
	parts  []legacyCorrectionPart
}

// readLegacyPositions adapts active compatibility segments to economic sources.
// Recognition refers back to its original backing, so its recording time cannot
// promote old funding ahead of a more recent purchase.
func (c *accrualCorrector) readLegacyPositions(ctx context.Context, input CorrectCollectedAccruedInput, source collectedSource, segments []legacylineage.Segment, used map[string]alpacadecimal.Decimal) ([]correctionPosition, map[string]legacyCorrectionEvidence, error) {
	if source.advanceReceivableIssueTransaction == nil {
		return c.readLegacyFundedPositions(ctx, input, source, segments, used)
	}
	positions := make(map[string]correctionPosition)
	evidence := make(map[string]legacyCorrectionEvidence)
	for _, segment := range segments {
		if err := segment.Validate(); err != nil {
			return nil, nil, err
		}
		state, backingID := segment.State, lo.FromPtr(segment.BackingTransactionGroupID)
		recognized := state == creditrealization.LineageSegmentStateEarningsRecognized
		if recognized {
			state, backingID = *segment.SourceState, lo.FromPtr(segment.SourceBackingTransactionGroupID)
		}
		key := backingID
		if state == creditrealization.LineageSegmentStateAdvanceUncovered {
			key = unknownOriginSource
		}
		if key == "" {
			return nil, nil, fmt.Errorf("legacy advance is missing backing evidence")
		}
		position, exists := positions[key]
		if !exists {
			position = correctionPosition{id: key, uncovered: key == unknownOriginSource}
			if !position.uncovered {
				group, err := c.ledger.GetTransactionGroup(ctx, models.NamespacedID{Namespace: input.Namespace, ID: backingID})
				if err != nil {
					return nil, nil, err
				}
				tx, err := backfillTransactionForSource(backfillTransactionForSourceInput{group: group, original: source.transaction, accountType: ledger.AccountTypeCustomerAccrued, templateCode: transactions.TemplateCode(transactions.TranslateCustomerAccruedCostBasisTemplate{})})
				if err != nil {
					return nil, nil, err
				}
				if tx == nil {
					return nil, nil, fmt.Errorf("legacy backing has no matching accrued translation")
				}
				position.recordedAt = tx.Cursor().CreatedAt
				position.orderKey = tx.ID().ID
			}
		}
		if recognized {
			position.earnings = position.earnings.Add(segment.Amount)
		} else {
			position.accrued = position.accrued.Add(segment.Amount)
		}
		positions[key] = position
		e := evidence[key]
		e.source = source
		e.parts = append(e.parts, legacyCorrectionPart{segment: segment, amount: segment.Amount})
		evidence[key] = e
	}
	return lo.Values(positions), evidence, nil
}

// Old funded roots can collapse several original credit entries. Split them
// back along those immutable entries and intersect recognition with its actual
// accrued routes before presenting positions to the common planner.
func (c *accrualCorrector) readLegacyFundedPositions(ctx context.Context, input CorrectCollectedAccruedInput, source collectedSource, segments []legacylineage.Segment, used map[string]alpacadecimal.Decimal) ([]correctionPosition, map[string]legacyCorrectionEvidence, error) {
	entries := slices.Clone(source.entries)
	slices.SortStableFunc(entries, compareCollectedFBOCorrectionSourceEntries)
	positions := make([]correctionPosition, len(entries))
	evidence := make(map[string]legacyCorrectionEvidence)
	capacity := make(map[string]alpacadecimal.Decimal)
	for i, entry := range entries {
		id := entry.ID().ID
		positions[i] = correctionPosition{id: id, order: i}
		narrowed := source
		narrowed.entries = []ledger.Entry{entry}
		evidence[id] = legacyCorrectionEvidence{source: narrowed}
		capacity[id] = entry.Amount().Abs().Sub(used[id])
	}
	// Reserve recognition references once even when several segments share a group.
	recognitionUsed := make(map[string]alpacadecimal.Decimal)
	for _, segment := range segments {
		if err := segment.Validate(); err != nil {
			return nil, nil, err
		}
		if segment.State != creditrealization.LineageSegmentStateEarningsRecognized {
			continue
		}
		group, err := c.ledger.GetTransactionGroup(ctx, models.NamespacedID{Namespace: input.Namespace, ID: *segment.BackingTransactionGroupID})
		if err != nil {
			return nil, nil, err
		}
		tx, err := c.forwardTransactionByTemplate(group, transactions.TemplateCode(transactions.RecognizeEarningsFromAttributableAccruedTemplate{}))
		if err != nil {
			return nil, nil, err
		}
		remaining := segment.Amount
		for i, entry := range entries {
			for _, recognized := range tx.Entries() {
				if recognized.PostingAddress().AccountType() != ledger.AccountTypeCustomerAccrued || !recognized.Amount().IsNegative() || !legacyRecognitionMatchesSource(source, entry, recognized) {
					continue
				}
				id := entry.ID().ID
				available := recognized.Amount().Abs().Sub(used[recognized.ID().ID]).Sub(recognitionUsed[recognized.ID().ID])
				take := minDecimal(remaining, minDecimal(capacity[id], available))
				if !take.IsPositive() {
					continue
				}
				positions[i].earnings = positions[i].earnings.Add(take)
				e := evidence[id]
				e.parts = append(e.parts, legacyCorrectionPart{segment: segment, amount: take})
				evidence[id] = e
				capacity[id] = capacity[id].Sub(take)
				recognitionUsed[recognized.ID().ID] = recognitionUsed[recognized.ID().ID].Add(take)
				remaining = remaining.Sub(take)
			}
		}
		if remaining.IsPositive() {
			return nil, nil, fmt.Errorf("legacy recognized segment exceeds matching original source evidence")
		}
	}
	for _, segment := range segments {
		if segment.State == creditrealization.LineageSegmentStateEarningsRecognized {
			continue
		}
		remaining := segment.Amount
		for i, entry := range entries {
			id := entry.ID().ID
			take := minDecimal(remaining, capacity[id])
			if !take.IsPositive() {
				continue
			}
			if segment.State == creditrealization.LineageSegmentStateReceivableCoverage {
				positions[i].coverage = positions[i].coverage.Add(take)
			} else {
				positions[i].accrued = positions[i].accrued.Add(take)
			}
			e := evidence[id]
			e.parts = append(e.parts, legacyCorrectionPart{segment: segment, amount: take})
			evidence[id] = e
			capacity[id] = capacity[id].Sub(take)
			remaining = remaining.Sub(take)
		}
		if remaining.IsPositive() {
			return nil, nil, fmt.Errorf("legacy segment exceeds remaining collection source evidence")
		}
	}
	return positions, evidence, nil
}

func legacyRecognitionMatchesSource(source collectedSource, fbo, recognized ledger.Entry) bool {
	for _, entry := range source.transaction.Entries() {
		if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerAccrued || !entry.Amount().IsPositive() {
			continue
		}
		if lo.FromPtr(entry.SourceChargeID()) != lo.FromPtr(fbo.SourceChargeID()) || lo.FromPtr(entry.SpendChargeID()) != lo.FromPtr(fbo.SpendChargeID()) {
			continue
		}
		a, b := entry.PostingAddress().Route().Route(), fbo.PostingAddress().Route().Route()
		if !a.Currency.Equal(b.Currency) || lo.FromPtr(a.CostBasisCurrency) != lo.FromPtr(b.CostBasisCurrency) {
			continue
		}
		if a.CostBasis == nil || b.CostBasis == nil {
			if a.CostBasis != b.CostBasis {
				continue
			}
		} else if !a.CostBasis.Equal(*b.CostBasis) {
			continue
		}
		if correctionEntryKey(entry) == correctionEntryKey(recognized) {
			return true
		}
	}
	return false
}

// writeLegacyCorrection translates exact planner selections. State only chooses
// the posting mechanics within the selected source; it cannot reorder sources.
func (c *accrualCorrector) writeLegacyCorrection(ctx context.Context, input CorrectCollectedAccruedInput, selected []correctionSelection, evidence map[string]legacyCorrectionEvidence, used, selectedSegments map[string]alpacadecimal.Decimal) ([]plannedAction, error) {
	var actions []plannedAction
	for _, selection := range selected {
		e := evidence[selection.id]
		for _, recognized := range []bool{true, false} {
			remaining := selection.amount.Sub(selection.earnings)
			if recognized {
				remaining = selection.earnings
			}
			for _, part := range e.parts {
				if (part.segment.State == creditrealization.LineageSegmentStateEarningsRecognized) != recognized {
					continue
				}
				take := minDecimal(remaining, part.amount)
				if !take.IsPositive() {
					continue
				}
				resolved, err := c.planSegmentCorrection(ctx, input, e.source, part.segment, take, used)
				if err != nil {
					return nil, err
				}
				actions = append(actions, resolved...)
				selectedSegments[part.segment.ID] = selectedSegments[part.segment.ID].Add(take)
				remaining = remaining.Sub(take)
			}
			if remaining.IsPositive() {
				return nil, fmt.Errorf("legacy selection exceeds its source evidence")
			}
		}
	}
	return actions, nil
}

// Pre-lineage collections have only their original entries and correction links.
// They use the same source-order planner without inventing downstream evidence.
func (c *accrualCorrector) planUntrackedCorrection(source collectedSource, amount alpacadecimal.Decimal, used map[string]alpacadecimal.Decimal) ([]plannedAction, error) {
	entries := slices.Clone(source.entries)
	slices.SortStableFunc(entries, compareCollectedFBOCorrectionSourceEntries)
	var positions []correctionPosition
	byID := make(map[string]ledger.Entry)
	for order, entry := range entries {
		id := entry.ID().ID
		positions = append(positions, correctionPosition{id: id, order: order, accrued: entry.Amount().Abs().Sub(used[id])})
		byID[id] = entry
	}
	selected, err := planCollectionCorrection(collectionCorrectionInput{amount: amount, positions: positions})
	if err != nil {
		return nil, err
	}
	var actions []plannedAction
	for _, selection := range selected {
		narrowed := source
		narrowed.entries = []ledger.Entry{byID[selection.id]}
		planned, err := plannedSourceCorrectionActions(narrowed, selection.amount, source.advanceReceivableIssueTransaction != nil, used)
		if err != nil {
			return nil, err
		}
		actions = append(actions, planned...)
	}
	return actions, nil
}
