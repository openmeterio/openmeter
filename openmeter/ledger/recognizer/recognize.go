package recognizer

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
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
			Currency:   in.Currency.Reference(),
		})
		if err != nil {
			return RecognizeEarningsResult{}, fmt.Errorf("load lineages: %w", err)
		}

		// Identify segments eligible for recognition, ordered deterministically by lineage ID.
		eligible := collectEligibleLineages(lineages)
		if len(eligible) == 0 {
			return RecognizeEarningsResult{}, nil
		}

		sources, allocations, err := s.planRecognition(ctx, in, eligible)
		if err != nil {
			return RecognizeEarningsResult{}, fmt.Errorf("plan recognition: %w", err)
		}
		actualAmount := alpacadecimal.Zero
		for _, allocation := range allocations {
			actualAmount = actualAmount.Add(allocation.amount)
		}
		if !actualAmount.IsPositive() {
			return RecognizeEarningsResult{}, nil
		}

		// Resolve postings for the exact accrued slices selected for these segments.
		resolved, err := transactions.ResolveTransactions(
			ctx,
			s.deps,
			transactions.ResolutionScope{
				CustomerID: in.CustomerID,
				Namespace:  in.CustomerID.Namespace,
			},
			transactions.RecognizeEarningsFromAttributableAccruedTemplate{
				At:       in.At,
				Amount:   actualAmount,
				Sources:  sources,
				Currency: in.Currency.Reference(),
			},
		)
		if err != nil {
			return RecognizeEarningsResult{}, fmt.Errorf("resolve recognition: %w", err)
		}
		if len(resolved) == 0 {
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

		// Persist exactly the segments that supplied the committed postings.
		if err := s.allocateRecognition(ctx, allocations, groupID, in.At); err != nil {
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

		for _, seg := range l.Segments {
			if recognizableSegmentStates[seg.State] && seg.Amount.IsPositive() {
				segments = append(segments, seg)
			}
		}

		if len(segments) > 0 {
			out = append(out, lineageEligible{
				lineage:  l,
				segments: segments,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].lineage.ID < out[j].lineage.ID
	})

	return out
}

// allocateRecognition transitions the preselected source segments atomically with
// their journal postings and preserves the backing needed by correction.
func (s *service) allocateRecognition(ctx context.Context, allocations []recognitionAllocation, groupID string, at time.Time) error {
	now := at.Truncate(time.Microsecond)
	for _, allocation := range allocations {
		seg := allocation.segment
		consumed := allocation.amount
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
	}
	return nil
}

func minDecimal(a, b alpacadecimal.Decimal) alpacadecimal.Decimal {
	if a.GreaterThan(b) {
		return b
	}

	return a
}
