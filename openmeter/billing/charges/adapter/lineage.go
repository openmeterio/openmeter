package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/creditrealizationlineage"
	"github.com/openmeterio/openmeter/openmeter/ent/db/creditrealizationlineagesegment"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func LoadActiveLineageSegments(
	ctx context.Context,
	db *entdb.Client,
	namespace string,
	realizationIDs []string,
) (map[string][]creditrealization.ActiveLineageSegment, error) {
	if len(realizationIDs) == 0 {
		return map[string][]creditrealization.ActiveLineageSegment{}, nil
	}

	lineages, err := db.CreditRealizationLineage.Query().
		Where(
			creditrealizationlineage.Namespace(namespace),
			creditrealizationlineage.RootRealizationIDIn(realizationIDs...),
		).
		WithSegments(func(q *entdb.CreditRealizationLineageSegmentQuery) {
			q.Where(creditrealizationlineagesegment.ClosedAtIsNil()).
				Order(creditrealizationlineagesegment.ByCreatedAt())
		}).
		All(ctx)
	if err != nil {
		return nil, err
	}

	return lo.SliceToMap(lineages, func(lineage *entdb.CreditRealizationLineage) (string, []creditrealization.ActiveLineageSegment) {
		return lineage.RootRealizationID, lo.Map(lineage.Edges.Segments, func(segment *entdb.CreditRealizationLineageSegment, _ int) creditrealization.ActiveLineageSegment {
			return creditrealization.ActiveLineageSegment{
				ID:                        segment.ID,
				Amount:                    segment.Amount,
				State:                     segment.State,
				BackingTransactionGroupID: segment.BackingTransactionGroupID,
			}
		})
	}), nil
}

func AttachInitialActiveLineageSegments(realizations creditrealization.Realizations) {
	for idx := range realizations {
		originKind, err := creditrealization.LineageOriginKindFromAnnotations(realizations[idx].Annotations)
		if err != nil {
			continue
		}

		initialState := creditrealization.InitialLineageSegmentState(originKind)
		if err := initialState.Validate(); err != nil {
			continue
		}

		realizations[idx].ActiveLineageSegments = []creditrealization.ActiveLineageSegment{
			{
				Amount: realizations[idx].Amount,
				State:  initialState,
			},
		}
	}
}

func LockAdvanceLineagesForBackfill(
	ctx context.Context,
	db *entdb.Client,
	namespace string,
	customerID string,
	currency currencyx.Code,
) ([]*entdb.CreditRealizationLineage, error) {
	if _, err := entutils.GetDriverFromContext(ctx); err != nil {
		return nil, fmt.Errorf("lock advance lineages for backfill must be called in a transaction: %w", err)
	}

	return db.CreditRealizationLineage.Query().
		Where(
			creditrealizationlineage.Namespace(namespace),
			creditrealizationlineage.CustomerIDEQ(customerID),
			creditrealizationlineage.CurrencyEQ(currency),
			creditrealizationlineage.HasSegmentsWith(
				creditrealizationlineagesegment.ClosedAtIsNil(),
				creditrealizationlineagesegment.StateEQ(creditrealization.LineageSegmentStateAdvanceUncovered),
			),
		).
		Order(creditrealizationlineage.ByCreatedAt()).
		ForUpdate().
		All(ctx)
}
