package recognizer

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/clock"
)

// recognizableSegmentStates are lineage segment states from which earnings can be recognized.
var recognizableSegmentStates = map[creditrealization.LineageSegmentState]bool{
	creditrealization.LineageSegmentStateRealCredit:        true,
	creditrealization.LineageSegmentStateAdvanceBackfilled: true,
}

// lineageEligible holds a lineage and its recognizable segment amounts.
type lineageEligible struct {
	lineage  lineage.Lineage
	segments []lineage.Segment
	amount   alpacadecimal.Decimal
}

func (s *service) RecognizeEarnings(ctx context.Context, in RecognizeEarningsInput) (RecognizeEarningsResult, error) {
	if err := in.Validate(); err != nil {
		return RecognizeEarningsResult{}, err
	}

	// Load all lineages for this customer+currency with their active segments.
	lineages, err := s.lnge.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
		Namespace:  in.CustomerID.Namespace,
		CustomerID: in.CustomerID.ID,
		Currency:   in.Currency,
	})
	if err != nil {
		return RecognizeEarningsResult{}, fmt.Errorf("load lineages: %w", err)
	}

	// Identify segments eligible for recognition, ordered deterministically by lineage ID.
	eligible := collectEligibleLineages(lineages)
	if len(eligible) == 0 {
		return RecognizeEarningsResult{}, nil
	}

	totalEligible := alpacadecimal.Zero
	for _, e := range eligible {
		totalEligible = totalEligible.Add(e.amount)
	}

	// Resolve the recognition template against the actual ledger accrued balance.
	resolved, err := transactions.ResolveTransactions(
		ctx,
		s.deps,
		transactions.ResolutionScope{
			CustomerID: in.CustomerID,
			Namespace:  in.CustomerID.Namespace,
		},
		transactions.RecognizeEarningsFromAttributableAccruedTemplate{
			At:       in.At,
			Amount:   totalEligible,
			Currency: in.Currency,
		},
	)
	if err != nil {
		return RecognizeEarningsResult{}, fmt.Errorf("resolve recognition: %w", err)
	}
	if len(resolved) == 0 {
		return RecognizeEarningsResult{}, nil
	}

	// Compute actual recognized amount from the template output entries.
	actualAmount := sumPositiveEntries(resolved)
	if !actualAmount.IsPositive() {
		return RecognizeEarningsResult{}, nil
	}

	// Commit the recognition to the ledger.
	group, err := s.ledger.CommitGroup(ctx, transactions.GroupInputs(
		in.CustomerID.Namespace,
		nil,
		resolved...,
	))
	if err != nil {
		return RecognizeEarningsResult{}, fmt.Errorf("commit recognition: %w", err)
	}

	groupID := group.ID().ID

	// Allocate actual recognized amount back to lineages in deterministic order
	// and transition their segments to earnings_recognized.
	if err := s.allocateRecognition(ctx, eligible, actualAmount, groupID); err != nil {
		rollbackErr := s.rollbackRecognition(ctx, in, group, actualAmount)
		if rollbackErr != nil {
			return RecognizeEarningsResult{}, fmt.Errorf("allocate recognition: %w; rollback recognition: %v", err, rollbackErr)
		}

		return RecognizeEarningsResult{}, fmt.Errorf("allocate recognition: %w (ledger recognition rolled back)", err)
	}

	return RecognizeEarningsResult{
		RecognizedAmount: actualAmount,
		LedgerGroupID:    groupID,
	}, nil
}

// collectEligibleLineages extracts lineages with recognizable active segments,
// sorted by lineage ID for deterministic ordering.
func collectEligibleLineages(lineages []lineage.Lineage) []lineageEligible {
	out := make([]lineageEligible, 0, len(lineages))

	for _, l := range lineages {
		var segments []lineage.Segment
		amount := alpacadecimal.Zero

		for _, seg := range l.Segments {
			if recognizableSegmentStates[seg.State] && seg.Amount.IsPositive() {
				segments = append(segments, seg)
				amount = amount.Add(seg.Amount)
			}
		}

		if amount.IsPositive() {
			out = append(out, lineageEligible{
				lineage:  l,
				segments: segments,
				amount:   amount,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].lineage.ID < out[j].lineage.ID
	})

	return out
}

// allocateRecognition distributes the actual recognized amount across eligible
// lineages and transitions their segments to earnings_recognized.
func (s *service) allocateRecognition(ctx context.Context, eligible []lineageEligible, actualAmount alpacadecimal.Decimal, groupID string) error {
	remaining := actualAmount
	now := clock.Now().Truncate(time.Microsecond)

	for _, e := range eligible {
		if !remaining.IsPositive() {
			break
		}

		lineageAlloc := minDecimal(e.amount, remaining)
		segRemaining := lineageAlloc

		for _, seg := range e.segments {
			if !segRemaining.IsPositive() {
				break
			}

			consumed := minDecimal(seg.Amount, segRemaining)

			if err := s.lnge.CloseSegment(ctx, seg.ID, now); err != nil {
				return fmt.Errorf("close segment %s: %w", seg.ID, err)
			}

			// If partial consumption, create remainder in original state.
			remainder := seg.Amount.Sub(consumed)
			if remainder.IsPositive() {
				if err := s.lnge.CreateSegment(ctx, lineage.CreateSegmentInput{
					LineageID:                 seg.LineageID,
					Amount:                    remainder,
					State:                     seg.State,
					BackingTransactionGroupID: seg.BackingTransactionGroupID,
				}); err != nil {
					return fmt.Errorf("create remainder segment: %w", err)
				}
			}

			// Create earnings_recognized segment for the consumed portion.
			sourceState := seg.State
			if err := s.lnge.CreateSegment(ctx, lineage.CreateSegmentInput{
				LineageID:                       seg.LineageID,
				Amount:                          consumed,
				State:                           creditrealization.LineageSegmentStateEarningsRecognized,
				BackingTransactionGroupID:       &groupID,
				SourceState:                     &sourceState,
				SourceBackingTransactionGroupID: seg.BackingTransactionGroupID,
			}); err != nil {
				return fmt.Errorf("create recognized segment: %w", err)
			}

			segRemaining = segRemaining.Sub(consumed)
		}

		remaining = remaining.Sub(lineageAlloc)
	}

	return nil
}

// rollbackRecognition compensates a committed recognition when lineage persistence fails.
func (s *service) rollbackRecognition(ctx context.Context, in RecognizeEarningsInput, group ledger.TransactionGroup, amount alpacadecimal.Decimal) error {
	tx, err := forwardRecognitionTransaction(group)
	if err != nil {
		return fmt.Errorf("find committed recognition transaction: %w", err)
	}

	correctionInputs, err := transactions.CorrectTransaction(ctx, s.deps, transactions.CorrectionInput{
		At:                  in.At,
		Amount:              amount,
		OriginalTransaction: tx,
		OriginalGroup:       group,
	})
	if err != nil {
		return fmt.Errorf("resolve recognition rollback: %w", err)
	}

	if len(correctionInputs) == 0 {
		return fmt.Errorf("resolve recognition rollback: no correction inputs")
	}

	if _, err := s.ledger.CommitGroup(ctx, transactions.GroupInputs(
		in.CustomerID.Namespace,
		nil,
		correctionInputs...,
	)); err != nil {
		return fmt.Errorf("commit recognition rollback: %w", err)
	}

	return nil
}

// sumPositiveEntries sums positive entry amounts across resolved transaction inputs.
// For recognition this is the total amount credited to earnings accounts.
func sumPositiveEntries(inputs []ledger.TransactionInput) alpacadecimal.Decimal {
	total := alpacadecimal.Zero

	for _, input := range inputs {
		for _, entry := range input.EntryInputs() {
			if entry.Amount().IsPositive() {
				total = total.Add(entry.Amount())
			}
		}
	}

	return total
}

func forwardRecognitionTransaction(group ledger.TransactionGroup) (ledger.Transaction, error) {
	templateName := transactions.TemplateName(transactions.RecognizeEarningsFromAttributableAccruedTemplate{})

	for _, tx := range group.Transactions() {
		name, err := ledger.TransactionTemplateNameFromAnnotations(tx.Annotations())
		if err != nil {
			continue
		}

		direction, err := ledger.TransactionDirectionFromAnnotations(tx.Annotations())
		if err != nil {
			continue
		}

		if name == templateName && direction == ledger.TransactionDirectionForward {
			return tx, nil
		}
	}

	return nil, fmt.Errorf("no forward recognition transaction found")
}

func minDecimal(a, b alpacadecimal.Decimal) alpacadecimal.Decimal {
	if a.GreaterThan(b) {
		return b
	}

	return a
}
