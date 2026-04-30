package creditrealization

import (
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/pkg/models"
)

const AnnotationLineageOriginKind = "billing.credit_realization.lineage_origin_kind"

type LineageOriginKind string

const (
	LineageOriginKindRealCredit LineageOriginKind = "real_credit"
	LineageOriginKindAdvance    LineageOriginKind = "advance"
)

func (k LineageOriginKind) Values() []string {
	return []string{
		string(LineageOriginKindRealCredit),
		string(LineageOriginKindAdvance),
	}
}

func (k LineageOriginKind) Validate() error {
	if !slices.Contains(k.Values(), string(k)) {
		return fmt.Errorf("invalid credit realization lineage origin kind: %s", k)
	}

	return nil
}

func LineageAnnotations(originKind LineageOriginKind) models.Annotations {
	return models.Annotations{
		AnnotationLineageOriginKind: string(originKind),
	}
}

func LineageOriginKindFromAnnotations(annotations models.Annotations) (LineageOriginKind, error) {
	originKind, ok := annotations.GetString(AnnotationLineageOriginKind)
	if !ok {
		return "", fmt.Errorf("missing credit realization lineage origin kind annotation")
	}

	out := LineageOriginKind(originKind)
	if err := out.Validate(); err != nil {
		return "", err
	}

	return out, nil
}

type LineageSegmentState string

const (
	// LineageSegmentStateRealCredit marks value that is still backed by the original
	// real-credit source and has not passed through advance/backfill flows.
	LineageSegmentStateRealCredit LineageSegmentState = "real_credit"
	// LineageSegmentStateAdvanceUncovered marks value that was collected as advance-backed
	// usage and is still not covered by a later credit purchase.
	LineageSegmentStateAdvanceUncovered LineageSegmentState = "advance_uncovered"
	// LineageSegmentStateAdvanceBackfilled marks value that was originally advance-backed
	// usage but was later covered by a credit purchase.
	LineageSegmentStateAdvanceBackfilled LineageSegmentState = "advance_backfilled"
	// LineageSegmentStateEarningsRecognized marks value that has been recognized as earnings
	// on the ledger (moved from accrued to earnings). BackingTransactionGroupID points to
	// the recognition ledger transaction group.
	LineageSegmentStateEarningsRecognized LineageSegmentState = "earnings_recognized"
)

func (s LineageSegmentState) Values() []string {
	return []string{
		string(LineageSegmentStateRealCredit),
		string(LineageSegmentStateAdvanceUncovered),
		string(LineageSegmentStateAdvanceBackfilled),
		string(LineageSegmentStateEarningsRecognized),
	}
}

func (s LineageSegmentState) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return fmt.Errorf("invalid credit realization lineage segment state: %s", s)
	}

	return nil
}

func InitialLineageSegmentState(originKind LineageOriginKind) LineageSegmentState {
	switch originKind {
	case LineageOriginKindRealCredit:
		return LineageSegmentStateRealCredit
	case LineageOriginKindAdvance:
		return LineageSegmentStateAdvanceUncovered
	default:
		return ""
	}
}
