package correction

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Template corrections targeting the same original transaction must merge before resolution.
type legacyCorrectionPlan struct {
	correctionPlan
	legacyCorrections []transactions.CorrectionInput
}

func (p *legacyCorrectionPlan) append(other legacyCorrectionPlan) {
	p.correctionPlan.append(other.correctionPlan)
	p.legacyCorrections = append(p.legacyCorrections, other.legacyCorrections...)
}

func (c *Corrector) planSegmentCorrection(ctx context.Context, input Input, source collectedSource, segment legacylineage.Segment, amount alpacadecimal.Decimal, used map[string]alpacadecimal.Decimal) (legacyCorrectionPlan, error) {
	// Legacy segment state chooses posting mechanics only. The shared planner
	// has already selected the funding source and amount.
	switch segment.State {
	case creditrealization.LineageSegmentStateRealCredit,
		creditrealization.LineageSegmentStateReceivableCoverage:
		return planSourceCorrection(input.AllocateAt, source, amount, used)
	case creditrealization.LineageSegmentStateAdvanceUncovered:
		return c.planLegacyAdvanceCorrection(ctx, input, source, nil, amount)
	case creditrealization.LineageSegmentStateAdvanceBackfilled:
		if segment.BackingTransactionGroupID == nil {
			return legacyCorrectionPlan{}, fmt.Errorf("advance_backfilled segment missing backing transaction group id")
		}

		return c.planLegacyAdvanceCorrection(ctx, input, source, segment.BackingTransactionGroupID, amount)
	case creditrealization.LineageSegmentStateEarningsRecognized:
		return c.planRecognizedEarningsSegment(ctx, input, source, segment, amount, used)
	default:
		return legacyCorrectionPlan{}, fmt.Errorf("unsupported active lineage segment state %s", segment.State)
	}
}

func (c *Corrector) planRecognizedEarningsSegment(ctx context.Context, input Input, source collectedSource, segment legacylineage.Segment, amount alpacadecimal.Decimal, used map[string]alpacadecimal.Decimal) (legacyCorrectionPlan, error) {
	if segment.BackingTransactionGroupID == nil || *segment.BackingTransactionGroupID == "" {
		return legacyCorrectionPlan{}, fmt.Errorf("earnings_recognized segment missing backing transaction group id")
	}

	if segment.SourceState == nil {
		return legacyCorrectionPlan{}, fmt.Errorf("earnings_recognized segment missing source state")
	}

	if *segment.SourceState == creditrealization.LineageSegmentStateEarningsRecognized {
		return legacyCorrectionPlan{}, fmt.Errorf("earnings_recognized segment source state cannot be earnings_recognized")
	}

	recognitionGroup, err := c.ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: input.Namespace,
		ID:        *segment.BackingTransactionGroupID,
	})
	if err != nil {
		return legacyCorrectionPlan{}, fmt.Errorf("get recognition transaction group %s: %w", *segment.BackingTransactionGroupID, err)
	}

	recognitionTx, err := c.forwardTransactionByTemplate(recognitionGroup, transactions.TemplateCode(transactions.RecognizeEarningsFromAttributableAccruedTemplate{}))
	if err != nil {
		return legacyCorrectionPlan{}, fmt.Errorf("find recognition transaction in group %s: %w", recognitionGroup.ID().ID, err)
	}

	sourceSegment := segment
	sourceSegment.State = *segment.SourceState
	sourceSegment.BackingTransactionGroupID = segment.SourceBackingTransactionGroupID
	sourceSegment.SourceState = nil
	sourceSegment.SourceBackingTransactionGroupID = nil

	sourcePlan, err := c.planSegmentCorrection(ctx, input, source, sourceSegment, amount, used)
	if err != nil {
		return legacyCorrectionPlan{}, err
	}

	entryAmounts, err := c.recognizedSourceAmounts(ctx, recognizedSourceAmountsInput{
		recognition: recognitionTx,
		plan:        sourcePlan,
		amount:      amount,
		used:        used,
	})
	if err != nil {
		return legacyCorrectionPlan{}, err
	}

	sourcePlan.legacyCorrections = append([]transactions.CorrectionInput{{
		At:                  input.AllocateAt,
		Amount:              amount,
		SourceEntryAmounts:  entryAmounts,
		OriginalTransaction: recognitionTx,
		OriginalGroup:       recognitionGroup,
	}}, sourcePlan.legacyCorrections...)

	return sourcePlan, nil
}

func planSourceCorrection(at time.Time, source collectedSource, amount alpacadecimal.Decimal, used map[string]alpacadecimal.Decimal) (legacyCorrectionPlan, error) {
	entryAmounts, err := reserveCorrectionSources(source.entries, amount, used)
	if err != nil {
		return legacyCorrectionPlan{}, err
	}

	return legacyCorrectionPlan{
		legacyCorrections: []transactions.CorrectionInput{{
			At:                  at,
			Amount:              amount,
			SourceEntryAmounts:  entryAmounts,
			OriginalTransaction: source.transaction,
			OriginalGroup:       source.group,
		}},
	}, nil
}
