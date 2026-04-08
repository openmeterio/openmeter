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
	LineageSegmentStateRealCredit        LineageSegmentState = "real_credit"
	LineageSegmentStateAdvanceUncovered  LineageSegmentState = "advance_uncovered"
	LineageSegmentStateAdvanceBackfilled LineageSegmentState = "advance_backfilled"
)

func (s LineageSegmentState) Values() []string {
	return []string{
		string(LineageSegmentStateRealCredit),
		string(LineageSegmentStateAdvanceUncovered),
		string(LineageSegmentStateAdvanceBackfilled),
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
