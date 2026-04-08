package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	chargesadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/creditrealizationlineagesegment"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) BackfillAdvanceLineageSegments(ctx context.Context, input creditpurchase.BackfillAdvanceLineageSegmentsInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		return tx.backfillAdvanceLineageSegments(ctx, input)
	})
}

func (a *adapter) backfillAdvanceLineageSegments(ctx context.Context, input creditpurchase.BackfillAdvanceLineageSegmentsInput) error {
	lineages, err := chargesadapter.LockAdvanceLineagesForBackfill(ctx, a.db, input.Namespace, input.CustomerID, input.Currency)
	if err != nil {
		return fmt.Errorf("lock advance lineages for backfill: %w", err)
	}

	if len(lineages) == 0 {
		return nil
	}

	lineageIDs := lo.Map(lineages, func(lineage *entdb.CreditRealizationLineage, _ int) string {
		return lineage.ID
	})

	segments, err := a.db.CreditRealizationLineageSegment.Query().
		Where(
			creditrealizationlineagesegment.ClosedAtIsNil(),
			creditrealizationlineagesegment.StateEQ(creditrealization.LineageSegmentStateAdvanceUncovered),
			creditrealizationlineagesegment.LineageIDIn(lineageIDs...),
		).
		Order(creditrealizationlineagesegment.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return fmt.Errorf("query active uncovered advance lineage segments: %w", err)
	}

	now := clock.Now().Truncate(time.Microsecond)
	remaining := input.Amount

	for _, segment := range segments {
		if !remaining.IsPositive() {
			break
		}

		coveredAmount := minDecimal(segment.Amount, remaining)

		if _, err := a.db.CreditRealizationLineageSegment.UpdateOneID(segment.ID).
			SetClosedAt(now).
			Save(ctx); err != nil {
			return fmt.Errorf("close uncovered advance lineage segment %s: %w", segment.ID, err)
		}

		remainder := segment.Amount.Sub(coveredAmount)
		if remainder.IsPositive() {
			if _, err := a.db.CreditRealizationLineageSegment.Create().
				SetID(ulid.Make().String()).
				SetLineageID(segment.LineageID).
				SetAmount(remainder).
				SetState(creditrealization.LineageSegmentStateAdvanceUncovered).
				Save(ctx); err != nil {
				return fmt.Errorf("create uncovered advance lineage remainder for segment %s: %w", segment.ID, err)
			}
		}

		if _, err := a.db.CreditRealizationLineageSegment.Create().
			SetID(ulid.Make().String()).
			SetLineageID(segment.LineageID).
			SetAmount(coveredAmount).
			SetState(creditrealization.LineageSegmentStateAdvanceBackfilled).
			SetBackingTransactionGroupID(input.BackingTransactionGroupID).
			Save(ctx); err != nil {
			return fmt.Errorf("create backfilled advance lineage segment for segment %s: %w", segment.ID, err)
		}

		remaining = remaining.Sub(coveredAmount)
	}

	return nil
}

func minDecimal(a, b alpacadecimal.Decimal) alpacadecimal.Decimal {
	if a.GreaterThan(b) {
		return b
	}

	return a
}
