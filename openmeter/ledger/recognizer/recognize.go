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
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
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

	return transaction.Run(ctx, s.transactionManager, func(ctx context.Context) (RecognizeEarningsResult, error) {
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

		// Commit the recognition to the ledger. The ledger joins the transaction
		// already carried by ctx, so lineage and ledger state commit atomically.
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
		if err := s.allocateRecognition(ctx, eligible, actualAmount, groupID, in.At); err != nil {
			return RecognizeEarningsResult{}, fmt.Errorf("allocate recognition: %w", err)
		}

		return RecognizeEarningsResult{
			RecognizedAmount: actualAmount,
			LedgerGroupID:    groupID,
		}, nil
	})
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
func (s *service) allocateRecognition(ctx context.Context, eligible []lineageEligible, actualAmount alpacadecimal.Decimal, groupID string, at time.Time) error {
	remaining := actualAmount
	now := at.Truncate(time.Microsecond)

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

			// Close the source segment before recreating its remainder and
			// recognized portion. This keeps the active segment set non-overlapping.
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
			// Source fields let correction unwind recognition back to the prior state.
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

func minDecimal(a, b alpacadecimal.Decimal) alpacadecimal.Decimal {
	if a.GreaterThan(b) {
		return b
	}

	return a
}
